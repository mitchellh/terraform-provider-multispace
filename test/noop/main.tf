terraform {
  required_providers {
    random = {
      source = "hashicorp/random"
      version = "3.1.0"
    }
  }
}

resource "random_pet" "pet" {}

output "pet" {
  value = random_pet.pet.id
}
