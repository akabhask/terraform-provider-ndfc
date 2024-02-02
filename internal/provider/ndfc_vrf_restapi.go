// Copyright © 2023 Cisco Systems, Inc. and its affiliates.
// All rights reserved.
//
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/terraform-provider-ndfc/internal/provider/helpers"
	"github.com/tidwall/gjson"
)

func (client *NdfcClient) ndfcVrfCreate(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, v *VRF) bool {
	var diags diag.Diagnostics
	var res gjson.Result
	var err error
	logit()
	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Create", v.Id.ValueString()))
	body := v.toBody(ctx)
	res, err, diags = client.ndfcRestApiRequest(ctx, "POST", v.getPath(), body)
	if err != nil {
		if strings.Contains(res.String(), "already exists") {
			// if entity already exists, continue with operation
			_, err, diags = client.ndfcRestApiRequest(ctx, "PUT", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), body)
			if err != nil {
				if ndfcCheckDiags(diags, resp) {
					tflog.Debug(ctx, fmt.Sprintf("Failed to post vrf %v path %v", v.getPath(), body))
				}
			}
		} else {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to post vrf %v path %v", v.getPath(), body))
			}
			return failed
		}
	}

	v.Id = types.StringValue(v.FabricName.ValueString() + "/" + v.VrfName.ValueString())
	if len(v.Attachments) > 0 {
		// deploy
		diags = client.ndfcPerSwitchAttachmentAndDeploy(ctx, v, "DEPLOYED")
		if ndfcCheckDiags(diags, resp) {
			return failed
		}
	}
	res, err, diags = client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to get vrf %v", v.VrfName.ValueString()))
		}
		return failed
	}
	v.fromBody(ctx, res)
	tflog.Debug(ctx, fmt.Sprintf("%s: Create finished successfully", v.Id.ValueString()))
	return success

}
func (client *NdfcClient) ndfcVrfRead(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, v *VRF) bool {

	logit()
	res, err, _ := client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), "")

	if err != nil {
		if strings.Contains(err.Error(), "StatusCode 400") || strings.Contains(err.Error(), "StatusCode 500") {
			resp.State.RemoveResource(ctx)
			return failed
		}
	}
	v.fromBody(ctx, res)
	res, err, diags := client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to get vrf %v", v.VrfName.ValueString()))
			return failed
		}
	}
	v.fromBodyAttachments(ctx, res, false)

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", v.Id.ValueString()))
	return success

}

func (client *NdfcClient) ndfcVrfUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, v *VRF, delete_attachments bool) bool {

	//var Attachments gjson.Result
	body := v.toBody(ctx)
	res, err, diags := client.ndfcRestApiRequest(ctx, "PUT", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), body)
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to (PUT)object , got error: %s, %s", err, res.String()))
			return failed
		}
	}

	log.Printf("Akash delete_attachments v.attachments %v %v", delete_attachments, len(v.Attachments))
	if !delete_attachments {
		if len(v.Attachments) > 0 {
			// deploy
			diags = client.ndfcPerSwitchAttachmentAndDeploy(ctx, v, "DEPLOYED")
			if ndfcCheckDiags(diags, resp) {
				return failed
			}
		}
	} else {
		// Undeploy
		Attachments, err, diags := client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to (GET)object , got error: %s, %s", err, res.String()))
				return failed
			}
		}
		if len(Attachments.Get("0").Array()) > 0 {
			diags = client.ndfcPerSwitchAttachmentAndDeploy(ctx, v, "NA")
			if ndfcCheckDiags(diags, resp) {
				return failed
			}
		}
	}

	res, err, diags = client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to retrieve object (GET), got error: %s, %s", err, res.String()))
			return failed
		}
	}
	v.fromBody(ctx, res)
	return success

}

func (client *NdfcClient) ndfcVrfDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, v *VRF) bool {
	var diags diag.Diagnostics
	var res gjson.Result
	var err error
	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Delete", v.Id.ValueString()))
	logit()
	log.Printf("Akash Beggining Delete")
	if len(v.Attachments) > 0 {
		diags = client.ndfcPerSwitchAttachmentAndDeploy(ctx, v, "NA")
		if ndfcCheckDiags(diags, resp) {
			return failed
		}
	}
	time.Sleep(5 * time.Second)
	res, err, diags = client.ndfcRestApiRequest(ctx, "DELETE", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to (DELETE) object , got error: %s, %s", err, res.String()))
			return failed
		}

	}
	tflog.Debug(ctx, fmt.Sprintf("%s: Delete finished successfully", v.Id.ValueString()))
	return success
}
func (client *NdfcClient) ndfcAttachSwitchToVrf(ctx context.Context, v *VRF, desired_status string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var res gjson.Result
	var serial_nos string
	var forced_dettach bool

	Attachments, err, diags := client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("Failed to get attachments for vrf %v", v.VrfName.ValueString()))
		return serial_nos, diags
	}
	log.Printf("Akash Attachments %v", Attachments)
	Attachments.Get("0").ForEach(func(k, r gjson.Result) bool {
		log.Printf("Akash r %v", r)
		if desired_status == "NA" {
			// if case of delete/destroy attachment needs to be forced detached
			forced_dettach = true
		} else {
			forced_dettach = false
		}
		serial_number := r.Get("switchSerialNo").String()
		for _, item := range v.Attachments {
			log.Printf("Akash item.SerialNumber.ValueString() %v", item.SerialNumber.ValueString())
			log.Printf("Akash serial_number %v", serial_number)
			log.Printf("Akash forced_dettach %v", forced_dettach)
			bodyAttachments := v.toBodyAttachments(ctx, r, forced_dettach)
			log.Printf("Akash bodyAttachments %v", bodyAttachments)
			res, err, diags = client.ndfcRestApiRequest(ctx, "POST", v.getPath()+"attachments", bodyAttachments)
			if err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Failed to perform attachments for vrf %v, got error: %s, %s", v.VrfName.ValueString(), err, res.String()))
				tflog.Debug(ctx, fmt.Sprintf("Failed to post attachments for vrf %v", v.VrfName.ValueString()))
				return false
			}
			diags = helpers.CheckAttachmentResponse(ctx, res)
			if diags.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("ndfcCheckDiags failed  %v for CheckAttachmentResponse",
					v.VrfName.ValueString()))
				diags.AddError("Client Error", fmt.Sprintf("Failed to perform attachments for vrf %v, got error: %s, %s", v.VrfName.ValueString(), err, res.String()))
				return false
			}
			if !item.DeployConfig.IsNull() && !item.DeployConfig.IsUnknown() && item.DeployConfig.ValueBool() {
				serial_nos += item.SerialNumber.ValueString() + " "
				log.Printf("Akash serial_no %v", serial_nos)
			} else if forced_dettach {
				serial_nos += item.SerialNumber.ValueString() + " "
				log.Printf("Akash serial_no %v", serial_nos)
			}
		}
		return true
	})
	return serial_nos, diags
}
func (client *NdfcClient) ndfcPerSwitchAttachmentAndDeploy(ctx context.Context, v *VRF, desired_status string) diag.Diagnostics {

	var res gjson.Result
	var CurrentStatus string
	not_deployed_list := make(map[string]bool)
	var diags diag.Diagnostics

	serial_nos, diags := client.ndfcAttachSwitchToVrf(ctx, v, desired_status)
	if diags.HasError() {
		tflog.Debug(ctx, fmt.Sprintf("ndfcGetAttachmentsPerVrf failed for VRF %v",
			v.VrfName.ValueString()))
		return diags
	}
	serial_number := strings.Fields(serial_nos)
	log.Printf("Akash serial number in list: %v", serial_number)
	if len(serial_number) > 0 {
		for _, item := range serial_number {
			CurrentStatus, diags = client.ndfcGetAttachmentsPerVrf(ctx, *v, item)
			if diags.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("ndfcGetAttachmentsPerVrf failed for VRF %v",
					v.VrfName.ValueString()))
				return diags
			}
			log.Printf("Akash serial number in list: %v", item)
			if CurrentStatus != "DEPLOYED" {
				diags, not_deployed_list = client.Deploy(ctx, *v, item, desired_status)
				if diags.HasError() {
					tflog.Debug(ctx, fmt.Sprintf("ndfcCheckDiags failed  %v for CheckAttachmentResponse",
						v.VrfName.ValueString()))
					return diags
				}
			}
		}
	}
	serial_nos = ""
	for index, item := range not_deployed_list {
		if item {
			serial_nos += index + " "
		}
	}
	if len(serial_nos) > 0 {
		diags.AddError("Client Error", fmt.Sprintf("Failed to deploy config for switch %s for vrf %s, got error: %s", serial_nos, v.VrfName.ValueString(), res.Get("0.message").String()))
	}
	return diags
}
func (client *NdfcClient) ndfcGetAttachmentsPerVrf(ctx context.Context, v VRF, serial_number string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var res gjson.Result
	var err error
	var status string
	Attachments, err, diags := client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Failed to retrieve VRFs, got error: %s, %s", err, res.String()))
		log.Printf("Akash status %v", status)
		return status, diags
	}
	Attachments.Get("0").ForEach(func(k, r gjson.Result) bool {
		cur_serial_number := r.Get("switchSerialNo").String()
		log.Printf("Akash cur_serial_number %v ", cur_serial_number)
		if serial_number == cur_serial_number {
			status = r.Get("lanAttachState").String()
			log.Printf("Akash status %v", status)
			return false
		}
		return true
	})
	return status, diags
}
func (client *NdfcClient) ndfcCompareVrfAttachments(p VRF, s VRF) ([]VRFAttachments, []VRFAttachments) {
	var TempAdd, TempDel []VRFAttachments
	var is_equal bool
	for _, p_value := range p.Attachments {
		is_equal = false
		for _, s_value := range s.Attachments {
			is_equal = reflect.DeepEqual(p_value, s_value)
			if is_equal {
				break
			}
		}
		if !is_equal {
			TempAdd = append(TempAdd, p_value)
		}
	}
	for _, s_value := range p.Attachments {
		is_equal = false
		for _, p_value := range s.Attachments {
			is_equal = reflect.DeepEqual(p_value, s_value)
			if is_equal {
				break
			}
		}
		if !is_equal {
			TempDel = append(TempDel, s_value)
		}
	}
	return TempAdd, TempDel
}
