variable "monorepo" {
  type        = string
  description = "Multispace provider repository. This must be forked."
  default     = "mitchellh/terraform-provider-multispace"
}

variable "oauth_token" {
  type        = string
  description = <<DESC
The OAuth token for GitHub access. This is a personal access token (PAT)
that can access the monorepo variable. The permissions required are:

- repo
- admin:repo_hook
- user

Even though the multispace repository are public, this is required by
Terraform Cloud.
DESC
}
