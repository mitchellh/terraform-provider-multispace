package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"hostname": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["hostname"],
				},

				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: descriptions["token"],
				},

				"ssl_skip_verify": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: descriptions["ssl_skip_verify"],
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				"multispace_run": resourceRun(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(
		ctx context.Context,
		d *schema.ResourceData,
	) (interface{}, diag.Diagnostics) {
		hostname := d.Get("hostname").(string)
		token := d.Get("token").(string)
		insecure := d.Get("ssl_skip_verify").(bool)

		client, err := getClient(version, hostname, token, insecure)
		if err != nil {
			return nil, []diag.Diagnostic{{
				Severity: diag.Error,
				Summary:  err.Error(),
			}}
		}

		return client, nil
	}
}

var descriptions = map[string]string{
	"hostname": "The Terraform Enterprise hostname to connect to. Defaults to app.terraform.io.",
	"token": "The token used to authenticate with Terraform Enterprise. We recommend omitting\n" +
		"the token which can be set as credentials in the CLI config file.",
	"ssl_skip_verify": "Whether or not to skip certificate verifications.",
}
