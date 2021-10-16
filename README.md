# Terraform Multispace Provider

The `multispace` Terraform provider implements resources to help work
with multi-workspace workflows in Terraform Cloud (or Enterprise) with
pure Terraform. The goal of the provider is to make it easy to create
and destroy full trees of Terraform workspaces used to represent a single
environment.

**Warning:** Despite my affiliation with HashiCorp, this is **NOT** an official
HashiCorp project and is not supported by HashiCorp. This was created on
my personal time for personal use cases.

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

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.15

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
