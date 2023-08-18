resource "ndfc_network" "example" {
  fabric_name                = "CML"
  network_name               = "NET1"
  display_name               = "NET1"
  network_template           = "Default_Network_Universal"
  network_extension_template = "Default_Network_Extension_Universal"
  vrf_name                   = "VRF1"
  gateway_ipv4_address       = "192.0.2.1/24"
  vlan_id                    = 1500
  gateway_ipv6_address       = "2001:db8::1/64,2001:db9::1/64"
  layer2_only                = false
  arp_suppression            = false
  ingress_replication        = false
  multicast_group            = "233.1.1.1"
  dhcp_relay_servers = [
    {
      address = "2.3.4.5"
      vrf     = "VRF1"
    }
  ]
  dhcp_relay_loopback_id = 134
  vlan_name              = "VLANXXX"
  interface_description  = "My int description"
  mtu                    = 9200
  loopback_routing_tag   = 11111
  trm                    = true
  secondary_gateway_1    = "192.168.2.1/24"
  secondary_gateway_2    = "192.168.3.1/24"
  secondary_gateway_3    = "192.168.4.1/24"
  secondary_gateway_4    = "192.168.5.1/24"
  route_target_both      = true
  netflow                = false
  svi_netflow_monitor    = "MON1"
  vlan_netflow_monitor   = "MON1"
  l3_gatway_border       = true
  attachments = [
    {
      serial_number       = "9DBYO6WQJ46"
      attach_switch_ports = "Ethernet1/10,Ethernet1/11"
      vlan_id             = 2010
    }
  ]
}
