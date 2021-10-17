package provider

import (
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
			},
		},
	})
}

const testAccResourceRun = `
resource "multispace_run" "root" {
  organization = "multispace-test"
  workspace    = "root"
}

resource "multispace_run" "A" {
  organization = "multispace-test"
  workspace    = "A"
  depends_on   = [multispace_run.root]
  manual_confirm = true
}

resource "multispace_run" "B" {
  organization = "multispace-test"
  workspace    = "B"
  depends_on   = [multispace_run.A]
}

resource "multispace_run" "C" {
  organization = "multispace-test"
  workspace    = "C"
  depends_on   = [multispace_run.A]
}
`
