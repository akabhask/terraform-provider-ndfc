terraform {
  required_providers {
    ndfc = {
      source = "registry.terraform.io/netascode/ndfc"
    }
  }
}
#provider "ndfc" {
#  username = "admin"
#  password = "admin!@#"
#  url      = "https://10.104.251.69"
#  retries = 4
#}
provider "ndfc" {
  username = "admin"
  password = "idgeR09!"
  url      = "https://rtp-ndfc1.cisco.com"
  retries = 4
}

resource "ndfc_vrf" "vrf1" {
  fabric_name                    = "Fabric-CL-Vegas24"
  vrf_name                       = "MyVRF_60055"
  vrf_template                   = "Default_VRF_Universal"
  vrf_extension_template         = "Default_VRF_Extension_Universal"
  vrf_id                         = 10022
  vlan_id                        = 1501
  vlan_name                      = "VLAN1501"
  interface_description          = "My int description"
  vrf_description                = "My vrf description"
  mtu                            = 9201
  loopback_routing_tag           = 11111
  redistribute_direct_route_map  = "FABRIC-RMAP-REDIST"
  max_bgp_paths                  = 2
  max_ibgp_paths                 = 3
  ipv6_link_local                = false
  trm                            = true
  no_rp                          = false
  rp_external                    = true
  rp_address                     = "1.2.3.4"
  rp_loopback_id                 = 100
  underlay_multicast_address     = "233.1.1.1"
  overlay_multicast_groups       = "234.0.0.0/8"
  mvpn_inter_as                  = false
  trm_bgw_msite                  = true
  advertise_host_routes          = true
  advertise_default_route        = false
  configure_static_default_route = false
  bgp_password                   = "1234567890ABCDEF"
  bgp_password_type              = "7"
  netflow                        = false
  netflow_monitor                = "MON1"
  disable_rt_auto                = true
  route_target_import            = "1:1"
  route_target_export            = "1:1"
  route_target_import_evpn       = "1:1"
  route_target_export_evpn       = "1:1"
  route_target_import_cloud_evpn = "1:1"
  route_target_export_cloud_evpn = "1:1"
  attachments = [
      {
       serial_number = "9XVYGLIE1U7"
       vlan_id       = 1300
       deploy_config = true
     },
     {
       serial_number = "919AEOOF7RV"
       vlan_id       = 2000
       deploy_config = true
     },
  ]
  timeouts = {
    create = "1m"
    delete = "1m"
    update = "1m"
    read = "1m"
  }
}

