package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRun(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRun,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"multispace_run.foo", "sample_attribute", regexp.MustCompile("^ba")),
				),
			},
		},
	})
}

const testAccResourceRun = `
resource "multispace_run" "foo" {
  organization = "mitchellh-mail"
  workspace    = "tfc"
}
`
