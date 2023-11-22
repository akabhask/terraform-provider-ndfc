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
	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Create", v.Id.ValueString()))
	// create vrf
	body := v.toBody(ctx)
	_, err, diags = client.ndfcRestApiRequest(ctx, "POST", v.getPath(), body)
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to post vrf %v path %v", v.getPath(), body))
		}
		return failed
	}

	v.Id = types.StringValue(v.FabricName.ValueString() + "/" + v.VrfName.ValueString())
	if len(v.Attachments) > 0 {
		// attach
		res, err, diags = client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to get attachments for vrf %v", v.VrfName.ValueString()))
			}
			return failed
		}
		bodyAttachments := v.toBodyAttachments(ctx, res)
		res, err, diags = client.ndfcRestApiRequest(ctx, "POST", v.getPath()+"attachments", bodyAttachments)
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to post attachments for vrf %v", v.VrfName.ValueString()))
			}
			return failed
		}
		diags = helpers.CheckAttachmentResponse(ctx, res)
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("ndfcCheckDiags failed  %v for CheckAttachmentResponse",
				v.VrfName.ValueString()))
			return failed
		}
		// deploy
		if DeployConfig {
			diags = client.Deploy(ctx, *v, "DEPLOYED")
			if ndfcCheckDiags(diags, resp) {
				return failed
			}
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

func (client *NdfcClient) ndfcVrfUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, v *VRF) bool {

	body := v.toBody(ctx)
	res, err, diags := client.ndfcRestApiRequest(ctx, "PUT", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), body)
	if err != nil {
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to (PUT)object , got error: %s, %s", err, res.String()))
			return failed
		}
	}

	if len(v.Attachments) > 0 {
		//  attach if attachements are present in config file
		res, err, diags = client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to retrieve object (GET), got error: %s, %s", err, res.String()))
				return failed
			}
		}
		bodyAttachments := v.toBodyAttachments(ctx, res)
		res, err, diags := client.ndfcRestApiRequest(ctx, "POST", v.getPath()+"attachments", bodyAttachments)
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to post for attachment: VRF %v, body %v", v.VrfName.ValueString(), bodyAttachments))
				return failed
			}
		}
		diags = helpers.CheckAttachmentResponse(ctx, res)
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("CheckAttachmentResponse failed for vrf %v", v.VrfName.ValueString()))
			return failed
		}

		time.Sleep(5 * time.Second)
		// deploy
		diags = client.Deploy(ctx, *v, "DEPLOYED")
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Deployment faild for vrf %v", v.VrfName.ValueString()))
			return failed
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
	// delete vrf
	if len(v.Attachments) > 0 {
		// detach everything
		res, err, diags = client.ndfcRestApiRequest(ctx, "GET", fmt.Sprintf("%vattachments?vrf-names=%v", v.getPath(), v.VrfName.ValueString()), "")
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to retrieve object (GET), got error: %s, %s", err, res.String()))
				return failed
			}
		}
		v.Attachments = make([]VRFAttachments, 0)
		bodyAttachments := v.toBodyAttachments(ctx, res)
		res, err, diags = client.ndfcRestApiRequest(ctx, "POST", v.getPath()+"attachments", bodyAttachments)
		if err != nil {
			if ndfcCheckDiags(diags, resp) {
				tflog.Debug(ctx, fmt.Sprintf("Failed to (POST) object , got error: %s, %s , bodyAttachments: %v", err, res.String(), bodyAttachments))
				return failed
			}
		}
		diags = helpers.CheckAttachmentResponse(ctx, res)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return failed
		}
	}
	diags = client.Deploy(ctx, *v, "NA")
	if ndfcCheckDiags(diags, resp) {
		return failed
	}
	time.Sleep(5 * time.Second)

	// delete vrf
	res, err, diags = client.ndfcRestApiRequest(ctx, "DELETE", fmt.Sprintf("%v%v", v.getPath(), v.VrfName.ValueString()), "")
	if err != nil {
		log.Printf("Delete failed ")
		if ndfcCheckDiags(diags, resp) {
			tflog.Debug(ctx, fmt.Sprintf("Failed to (DELETE) object , got error: %s, %s", err, res.String()))
			return failed
		}
	}
	log.Printf("Delete finished successfully ")
	tflog.Debug(ctx, fmt.Sprintf("%s: Delete finished successfully", v.Id.ValueString()))
	return success
}
