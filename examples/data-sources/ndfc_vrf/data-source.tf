terraform {
  required_providers {
    ndfc = {
      source = "registry.terraform.io/netascode/ndfc"
    }
  }
}
provider "ndfc" {
  username = "admin"
  password = "admin!@#"
  url      = "https://10.104.251.69"
  retries = 4
}
data "ndfc_vrf" "example" {
  fabric_name = "CML"
  vrf_name    = "VRF1"
  attachments = [{
  }]
}
