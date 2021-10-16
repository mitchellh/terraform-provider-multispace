terraform {
  required_providers {
    tfe = {
      source  = "hashicorp/tfe"
      version = "~> 0.26.1"
    }

    random = {
      source = "hashicorp/random"
      version = "3.1.0"
    }
  }
}

locals {
  # All the workspaces we will create. They all do the same thing,
  # which is run the "noop" module which does [almost] nothing.
  workspaces = ["root", "A", "B", "C"]
}

resource "random_string" "random" {
  length           = 8
  special          = false
}

resource "tfe_organization" "org" {
  name = "multispace-test-${random_string.random.string}"
}

resource "tfe_oauth_client" "client" {
  organization     = tfe_organization.org.name
  service_provider = "github"
  http_url         = "https://github.com"
  api_url          = "https://api.github.com"
  oauth_token      = var.oauth_token
}

resource "tfe_workspace" "ws" {
  for_each          = local.workspaces
  name              = each.value
  organization      = tfe_organization.org.name
  working_directory = "test/noop"
  queue_all_runs    = false

  vcs_repo {
    identifier     = var.github_repo
    oauth_token_id = tfe_oauth_client.client.oauth_token_id
  }
}
