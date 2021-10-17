# Terraform Multispace Provider

The `multispace` Terraform provider implements resources to help work
with multi-workspace workflows in Terraform Cloud (or Enterprise).
The goal of the provider is to make it easy to perform cascading
creation/deletes in the proper order across a series of dependent
Terraform workspaces.

For more details on motivation, see the ["why?" section](#why).

**Warning:** Despite my affiliation with HashiCorp, this is **NOT** an official
HashiCorp project and is not supported by HashiCorp. This was created on
my personal time for personal use cases.

## Features

  * Cascading create/destroy of multiple Terraform workspaces in
    dependency order.

  * Automatic retry of failed plans or applies within a workspace.

## Usage

The example below cascades applies and destroys across multiple workspaces.

The recommended usage includes pairing this with the
[tfe provider](https://registry.terraform.io/providers/hashicorp/tfe/latest).
The `tfe` provider is used to configure your workspaces, and the
`multispace` provider is used to create a tree of workspaces that
are initialized together.

**Note on usage:** I usually only use this to manage the create/destroy
lifecycle today. The steady-state modification workflow uses the standard
Terraform Cloud VCS-driven workflows. This provider just helps me stand up
my initial environments and subsequently tear them down.

```terraform
resource "multispace_run" "root" {
  # Use string workspace names here and not data sources so that
  # you can define the multispace runs before the workspace even exists.
  workspace = "tfc"
}

resource "multispace_run" "physical" {
  workspace = "k8s-physical"
  depends_on = [multispace_run.root]
}

resource "multispace_run" "core" {
  workspace = "k8s-core"
  depends_on = [multispace_run.physical]
}

resource "multispace_run" "dns" {
  workspace = "dns"
  depends_on = [multispace_run.root]
}

resource "multispace_run" "ingress" {
  workspace = "ingress"
  depends_on = [multispace_run.core, multispace_run.dns]
}
```

## Why?

Multiple [workspaces](https://www.terraform.io/docs/cloud/workspaces/index.html)
are my recommended approach to working with Terraform. Small, focused workspaces
make Terraform runs fast, limit the blast radius, and enable easier
work separation by teams. The [`terraform_remote_state` data source](https://www.terraform.io/docs/language/state/remote-state-data.html)
can be used to pass outputs from one workspace to another workspace. This
enables a clean separation of responsibilities. This is also
[officially recommended by Terraform](https://www.terraform.io/docs/cloud/guides/recommended-practices/part1.html).

I also use multiple workspaces as a way to model **environments**: dev,
staging, production, etc. An environment to me is a collection of many
workspaces working together to create a working environment. For example,
one project of mine has the following workspaces that depend on each other
to create a full environment: k8s-physical, k8s-core, dns, metrics, etc.

**The problem statement** is that I do not have a good way to create my
workspaces, create them all at once in the right order, and then destroy them
if I'm done with the environment. Without this provider, I have to manually
click through the Terraform Cloud UI.

With this provider, I can now create a single Terraform module that is used
to launch a _complete environment_ for a project, composed of multiple workspaces.
And I can destroy that entire environment with a `terraform destroy`, which
cascades a destroy through all the workspaces in the correct order thanks
to Terraform.

Note that Terraform Cloud does provide [run triggers](https://www.terraform.io/docs/cloud/workspaces/run-triggers.html)
but this doesn't quite solve my problem: I don't generally want run triggers,
I just want to mainly do what I'd describe as a "cascading apply/destroy"
for creation/destruction. For steady-state modifications once an environment
exists, I use the typical Terraform Cloud VCS-driven workflow (which may or
may not involve run triggers at that point).

## Future Functionality

The list below has functionality I'd like to add in the future:

  * Only create if there is state, otherwise, assume initialization is done.
    This will allow this provider to be adopted into existing workspace
    trees more easily.

  * Option per resource to wait for a manual plan confirmation. This makes
    this safer to run.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
