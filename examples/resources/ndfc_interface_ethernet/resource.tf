resource "ndfc_interface_ethernet" "example" {
  serial_number         = "9DBYO6WQJ46"
  interface_name        = "Ethernet1/13"
  policy                = "int_access_host"
  bpdu_guard            = "true"
  port_type_fast        = false
  mtu                   = "default"
  speed                 = "Auto"
  access_vlan           = 500
  interface_description = "My interface description"
  orphan_port           = false
  freeform_config       = "delay 200"
  admin_state           = false
  ptp                   = false
  netflow               = false
  allowed_vlans         = "10-20"
}
