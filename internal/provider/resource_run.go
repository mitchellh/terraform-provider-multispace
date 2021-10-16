package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRun() *schema.Resource {
	return &schema.Resource{
		Description: "Workspace run (create/destroy)",

		CreateContext: resourceRunCreate,
		ReadContext:   resourceRunRead,
		UpdateContext: resourceRunUpdate,
		DeleteContext: resourceRunDelete,

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceRunCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return doRun(ctx, d, meta, false)
}

func resourceRunRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*tfe.Client)
	id := d.Id()

	// Get our run. If it doesn't exist, then we assume that we were never
	// created. And if it exists in any form, we assume we're still in that
	// state.
	_, err := client.Runs.Read(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	return nil
}

func resourceRunUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// We have no mutable fields, this should never be called.
	return diag.Errorf("update should never be called")
}

func resourceRunDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return doRun(ctx, d, meta, true)
}

func doRun(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
	destroy bool,
) diag.Diagnostics {
	// We only set the ID on create
	setId := func(v string) {
		if !destroy {
			d.SetId(v)
		}
	}

	client := meta.(*tfe.Client)
	org := d.Get("organization").(string)
	workspace := d.Get("workspace").(string)

	// Get our workspace because we need it to queue a plan
	ws, err := client.Workspaces.Read(ctx, org, workspace)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create a run
	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		Message: tfe.String(fmt.Sprintf(
			"terraform-provider-multispace on %s",
			time.Now().Format("Mon Jan 2 15:04:05 MST 2006"),
		)),
		Workspace: ws,
		IsDestroy: tfe.Bool(destroy),

		// Never auto-apply because we handle all that.
		AutoApply: tfe.Bool(false),
	})

	// The ID we use is the run we queue. We can use this to look this
	// run up again in the case of a partial failure.
	setId(run.ID)
	log.Printf("[INFO] run created: %s", run.ID)

	// Wait for the plan to complete.
	run, diags := waitForRun(ctx, client, org, run, ws, true, []tfe.RunStatus{
		tfe.RunPlanned,
		tfe.RunPlannedAndFinished,
		tfe.RunErrored,
	}, []tfe.RunStatus{
		tfe.RunPending,
		tfe.RunPlanQueued,
		tfe.RunPlanning,
	})
	if diags != nil {
		return diags
	}

	// If the plan has no changes, then we're done.
	if !run.HasChanges || run.Status == tfe.RunPlannedAndFinished {
		log.Printf("[INFO] plan finished, no changes")
		return nil
	}

	// If the run errored, we should have exited already but lets just exit now.
	if run.Status == tfe.RunErrored {
		// Clear the ID, we didn't create anything.
		setId("")

		return diag.Errorf(
			"Run %q errored during plan. Please open the web UI to view the error",
			run.ID,
		)
	}

	// Apply the plan.
	// NOTE(mitchellh): The reason I don't just autoapply above is because
	// in the future it'd be nice to make an option that pauses for certain
	// workspaces so they can be verified manually.
	log.Printf("[INFO] plan complete, confirming apply. %q", run.ID)
	if err := client.Runs.Apply(ctx, run.ID, tfe.RunApplyOptions{
		Comment: tfe.String(fmt.Sprintf(
			"terraform-provider-multispace on %s",
			time.Now().Format("Mon Jan 2 15:04:05 MST 2006"),
		)),
	}); err != nil {
		return diag.FromErr(err)
	}

	// Wait now for the apply to complete
	run, diags = waitForRun(ctx, client, org, run, ws, false, []tfe.RunStatus{
		tfe.RunApplied,
		tfe.RunErrored,
	}, []tfe.RunStatus{
		tfe.RunConfirmed,
		tfe.RunApplyQueued,
		tfe.RunApplying,
	})
	if diags != nil {
		return diags
	}

	// If the run errored, we should have exited already but lets just exit now.
	if run.Status == tfe.RunErrored {
		// Clear the ID, we didn't create anything.
		setId("")

		return diag.Errorf(
			"Run %q errored during apply. Please open the web UI to view the error",
			run.ID,
		)
	}

	return nil
}
