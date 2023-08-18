resource "ndfc_interface_loopback" "example" {
  serial_number         = "9DBYO6WQJ46"
  interface_name        = "loopback123"
  policy                = "int_loopback"
  vrf                   = "VRF1"
  ipv4_address          = "5.6.7.8"
  ipv6_address          = "2001::10"
  route_map_tag         = "12346"
  interface_description = "My interface description"
  freeform_config       = "logging event port link-status"
  admin_state           = false
}
