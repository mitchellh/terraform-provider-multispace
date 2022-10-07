package provider

// Much of this implementation is taken from the core Terraform project,
// licensed MPL2. Slight modifications have been made.

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

var (
	backoffMin      = 1000.0
	backoffMax      = 3000.0
	runPollInterval = 3 * time.Second
)

// backoff will perform exponential backoff based on the iteration and
// limited by the provided min and max (in milliseconds) durations.
func backoff(min, max float64, iter int) time.Duration {
	backoff := math.Pow(2, float64(iter)/5) * min
	if backoff > max {
		backoff = max
	}
	return time.Duration(backoff) * time.Millisecond
}

func waitForRun(
	ctx context.Context,
	client *tfe.Client,
	orgName string,
	r *tfe.Run,
	w *tfe.Workspace,
	opPlan bool,
	terminal []tfe.RunStatus,
	progress []tfe.RunStatus,
) (*tfe.Run, diag.Diagnostics) {
	if progress == nil {
		progress = []tfe.RunStatus{tfe.RunPending, tfe.RunConfirmed}
	}

	started := time.Now()
	updated := started
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return r, diag.FromErr(ctx.Err())
		case <-time.After(backoff(backoffMin, backoffMax, i)):
			// Timer up, show status
		}

		// Retrieve the run to get its current status.
		r, err := client.Runs.Read(ctx, r.ID)
		if err != nil {
			return r, diag.Errorf("Failed to retrieve run: %s", err)
		}

		// If we have terminal states and we reached any, we're done.
		for _, s := range terminal {
			if r.Status == s {
				log.Printf("[DEBUG] reached terminal state %q", r.Status)
				return r, nil
			}
		}

		// If we have progressive states and we're not any of those, then
		// exit early.
		found := false
		for _, s := range progress {
			if r.Status == s {
				found = true
				break
			}
		}
		if !found {
			if r.Actions.IsConfirmable {
				log.Printf("[DEBUG] non-progressive is-confirmable state, exiting %q", r.Status)
				return r, nil
			}

			log.Printf("[DEBUG] non-progressive state, waiting %q", r.Status)
		}

		// Check if 30 seconds have passed since the last update.
		current := time.Now()
		if i == 0 || current.Sub(updated).Seconds() > 30 {
			updated = current
			position := 0
			elapsed := ""

			// Calculate and set the elapsed time.
			if i > 0 {
				elapsed = fmt.Sprintf(
					" (%s elapsed)", current.Sub(started).Truncate(30*time.Second))
			}

			// Retrieve the workspace used to run this operation in.
			w, err = client.Workspaces.ReadByID(ctx, w.ID)
			if err != nil {
				return nil, diag.Errorf("Failed to retrieve workspace: %s", err)
			}

			// If the workspace is locked the run will not be queued and we can
			// update the status without making any expensive calls.
			if w.Locked && w.CurrentRun != nil {
				cr, err := client.Runs.Read(ctx, w.CurrentRun.ID)
				if err != nil {
					return r, diag.Errorf("Failed to retrieve current run: %s", err)
				}
				if cr.Status == tfe.RunPending {
					log.Printf(
						"[DEBUG] Waiting for the manually locked workspace to " +
							"be unlocked..." + elapsed)
					continue
				}
			}

			// Skip checking the workspace queue when we are the current run.
			if w.CurrentRun == nil || w.CurrentRun.ID != r.ID {
				found := false
				options := tfe.RunListOptions{}
			runlist:
				for {
					rl, err := client.Runs.List(ctx, w.ID, options)
					if err != nil {
						return r, diag.Errorf("Failed to retrieve run list: %s", err)
					}

					// Loop through all runs to calculate the workspace queue position.
					for _, item := range rl.Items {
						if !found {
							if r.ID == item.ID {
								found = true
							}
							continue
						}

						// If the run is in a final state, ignore it and continue.
						switch item.Status {
						case tfe.RunApplied, tfe.RunCanceled, tfe.RunDiscarded, tfe.RunErrored:
							continue
						case tfe.RunPlanned:
							if opPlan {
								continue
							}
						}

						// Increase the workspace queue position.
						position++

						// Stop searching when we reached the current run.
						if w.CurrentRun != nil && w.CurrentRun.ID == item.ID {
							break runlist
						}
					}

					// Exit the loop when we've seen all pages.
					if rl.CurrentPage >= rl.TotalPages {
						break
					}

					// Update the page number to get the next page.
					options.PageNumber = rl.NextPage
				}

				if position > 0 {
					log.Printf(
						"[INFO] Waiting for %d run(s) to finish before being queued...%s",
						position,
						elapsed,
					)
					continue
				}
			}

			options := tfe.RunQueueOptions{}
		search:
			for {
				rq, err := client.Organizations.RunQueue(ctx, orgName, options)
				if err != nil {
					return r, diag.Errorf("Failed to retrieve queue: %s", err)
				}

				// Search through all queued items to find our run.
				for _, item := range rq.Items {
					if r.ID == item.ID {
						position = item.PositionInQueue
						break search
					}
				}

				// Exit the loop when we've seen all pages.
				if rq.CurrentPage >= rq.TotalPages {
					break
				}

				// Update the page number to get the next page.
				options.PageNumber = rq.NextPage
			}

			if position > 0 {
				c, err := client.Organizations.Capacity(ctx, orgName)
				if err != nil {
					return r, diag.Errorf("Failed to retrieve capacity: %s", err)
				}
				log.Printf(
					"[INFO] Waiting for %d queued run(s) to finish before starting...%s",
					position-c.Running,
					elapsed,
				)
				continue
			}

			log.Printf("[DEBUG] Waiting for the run to start...%s", elapsed)
		}
	}
}
