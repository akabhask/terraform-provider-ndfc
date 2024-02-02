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
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/terraform-provider-ndfc/internal/provider/helpers"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func logit() {
	f, err := os.OpenFile("logfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
}

var failed = false
var success = true

type NdfcUserTimeout interface {
	ndfcSetTimeOut()
}
type Any interface{}

// Get path for vrf resources
func (resource VRF) getPath() string {
	return fmt.Sprintf("/lan-fabric/rest/top-down/v2/fabrics/%v/vrfs/", url.QueryEscape(fmt.Sprintf("%v", resource.FabricName.ValueString())))
}
func ndfcCheckDiags(diags diag.Diagnostics, a Any) bool {
	switch resp := a.(type) {
	case *resource.CreateResponse:
		resp.Diagnostics.Append(diags...)
		return resp.Diagnostics.HasError()
	case *resource.UpdateResponse:
		resp.Diagnostics.Append(diags...)
		return resp.Diagnostics.HasError()
	case *resource.DeleteResponse:
		resp.Diagnostics.Append(diags...)
		return resp.Diagnostics.HasError()
	case *resource.ReadResponse:
		resp.Diagnostics.Append(diags...)
		return resp.Diagnostics.HasError()
	}
	return false
}

func (client *NdfcClient) ndfcRestApiRequest(ctx context.Context, requestType string, path string, payLoad string) (gjson.Result, error, diag.Diagnostics) {
	var res gjson.Result
	var err error
	var diags diag.Diagnostics
request_retry:
	client.updateMutex.Lock()
	switch requestType {
	case "GET":
		res, err = client.client.Get(path)
	case "POST":
		res, err = client.client.Post(path, payLoad)
	case "PUT":
		res, err = client.client.Put(path, payLoad)
	case "DELETE":
		res, err = client.client.Delete(path, "")
	default:
		tflog.Debug(ctx, fmt.Sprintf("request type not found : %v", requestType))
		err = errors.New("wrong request type")
	}
	client.updateMutex.Unlock()
	tflog.Debug(ctx, fmt.Sprintf(" ndfcRestApiRequest requesttype: %v path %v res : %v  err %v payload",
		requestType, path, res.String(), err))
	if err != nil {
		if strings.Contains(res.String(), "connect: operation timed out") {
			// Sometimes connection to NDFC fails, we retry till we connect
			tflog.Debug(ctx, res.String())
			time.Sleep(10 * time.Second)
			goto request_retry
		}
		if strings.Contains(err.Error(), "connection refused") {
			// Sometimes connection to NDFC fails, we retry till we connect
			tflog.Debug(ctx, res.String())
			time.Sleep(10 * time.Second)
			goto request_retry
		}
		diags.AddError("Client Error",
			fmt.Sprintf("Failed to perform operation (%s) got error: %s, %s", requestType, err, res.String()))
	}

	return res, err, diags
}

func (v *VRF) ndfcSetTimeOut(ctx context.Context, operation string) (context.Context, diag.Diagnostics) {
	var diags diag.Diagnostics
	var timeout time.Duration

	switch operation {
	case "CREATE":
		timeout, diags = v.Timeouts.Create(ctx, time.Minute)
	case "UPDATE":
		timeout, diags = v.Timeouts.Update(ctx, time.Minute)
	case "DELETE":
		timeout, diags = v.Timeouts.Delete(ctx, time.Minute)
	case "READ":
		timeout, diags = v.Timeouts.Read(ctx, time.Minute)
	default:
		tflog.Debug(ctx, fmt.Sprintf("operation not found : %v", operation))
		return ctx, diags
	}
	if !diags.HasError() {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return ctx, diags
	}
	return ctx, diags
}

func (client *NdfcClient) WaitForStatus(ctx context.Context, serial_number string, v VRF, expectedStatus string) string {
	var CurrentStatus string
	var diags diag.Diagnostics

	for i := 0; i < helpers.NDFC_CHECK_STATUS_RETRIES; i++ {
		time.Sleep(helpers.NDFC_CHECK_STATUS_DELAY * time.Second)
		CurrentStatus, diags = client.ndfcGetAttachmentsPerVrf(ctx, v, serial_number)
		if diags.HasError() {
			tflog.Debug(ctx, fmt.Sprintf("ndfcGetAttachmentsPerVrf failed for VRF %v",
				v.VrfName.ValueString()))
			return CurrentStatus
		}
		log.Printf("WaitForStatus Akash status string %v", CurrentStatus)
		tflog.Debug(ctx, fmt.Sprintf("WaitForStatus status: %v try: %v", CurrentStatus, i))
		if strings.Contains(expectedStatus, CurrentStatus) {
			return CurrentStatus
		}

	}
	return CurrentStatus
}
func (r *NdfcClient) checkStateStabilized(ctx context.Context, serial_number string, v VRF, expectedStatus string) string {
	var CurrentStatus string
	var diags diag.Diagnostics

	for i := 0; i < helpers.NDFC_CHECK_STATUS_RETRIES; i++ {
		time.Sleep(helpers.NDFC_CHECK_STATUS_DELAY * time.Second)
		CurrentStatus, diags = r.ndfcGetAttachmentsPerVrf(ctx, v, serial_number)
		if diags.HasError() {
			tflog.Debug(ctx, fmt.Sprintf("ndfcGetAttachmentsPerVrf failed for VRF %v",
				v.VrfName.ValueString()))
			return CurrentStatus
		}
		log.Printf("checkStateStabilized Akash status string %v %v %v", CurrentStatus, expectedStatus, i)
		tflog.Debug(ctx, fmt.Sprintf("checkExpectedState status: %v %v %v", CurrentStatus, expectedStatus, i))
		if !strings.Contains(expectedStatus, CurrentStatus) {
			return CurrentStatus
		}
	}
	return CurrentStatus
}

func (client *NdfcClient) Deploy(ctx context.Context, v VRF, serial_number string, expectedStatus string) (diag.Diagnostics, map[string]bool) {
	var diags diag.Diagnostics
	var CurrentStatus string
	var res gjson.Result
	not_deployed_list := make(map[string]bool)
	var err error

	NextValidState := "DEPLOYED OUT-OF-SYNC FAILED PENDING NA"
	logit()
	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Deploy", v.Id.ValueString()))
	body := ""
	body, _ = sjson.Set(body, serial_number, v.VrfName.ValueString())
	log.Printf("Akash deploy body %v", body)
	for i := 0; i < 2; i++ {
		CurrentStatus, diags = client.ndfcGetAttachmentsPerVrf(ctx, v, serial_number)
		if diags.HasError() {
			tflog.Debug(ctx, fmt.Sprintf("ndfcGetAttachmentsPerVrf failed for VRF %v",
				v.VrfName.ValueString()))
			not_deployed_list[serial_number] = true
			return diags, not_deployed_list
		}
		log.Printf("Akash CurrentStatus: %v serial_number %v", CurrentStatus, serial_number)
		if strings.Contains(CurrentStatus, "IN PROGRESS") {
			CurrentStatus = client.WaitForStatus(ctx, serial_number, v, NextValidState)
			if !strings.Contains(NextValidState, CurrentStatus) {
				diags.AddError("Client Error", fmt.Sprintf("unknown v: %v reached when trying to deploy",
					CurrentStatus))
				not_deployed_list[serial_number] = true
				return diags, not_deployed_list
			}
		}
		CurrentStatus = client.checkStateStabilized(ctx, serial_number, v, expectedStatus)
		log.Printf("Akash CurrentStatus after IN PROGRESS: %v", CurrentStatus)
		switch CurrentStatus {
		case "DEPLOYED":
			if expectedStatus == "DEPLOYED" {
				return diags, not_deployed_list
			}
		case "NA":
			if expectedStatus == "NA" {
				return diags, not_deployed_list
			}
		case "OUT-OF-SYNC":
			fallthrough
		case "PENDING":
			res, err, diags = client.ndfcRestApiRequest(ctx, "POST", "/lan-fabric/rest/top-down/vrfs/deploy", body)
			if err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Failed to POST, got error: %s, %s", err, res.String()))
				return diags, not_deployed_list
			}
		case "FAILED":
			not_deployed_list[serial_number] = true
			return diags, not_deployed_list
		default:
			diags.AddError("Client Error", fmt.Sprintf("Unknown state '%s' for serial number '%s' when trying to Deploy", CurrentStatus, serial_number))
			return diags, not_deployed_list
		}
	}
	NextValidState = "DEPLOYED OUT-OF-SYNC FAILED NA"

	CurrentStatus = client.WaitForStatus(ctx, serial_number, v, NextValidState)
	if !strings.Contains(NextValidState, CurrentStatus) {
		diags.AddError("Client Error", fmt.Sprintf("Reached state %v which is not expected",
			CurrentStatus))
		not_deployed_list[serial_number] = true
		return diags, not_deployed_list
	}
	if CurrentStatus == "FAILED" || CurrentStatus == "OUT-OF-SYNC" {
		not_deployed_list[serial_number] = true
		return diags, not_deployed_list
	}

	CurrentStatus = client.checkStateStabilized(ctx, serial_number, v, CurrentStatus)
	if !strings.Contains(NextValidState, CurrentStatus) {
		diags.AddError("Client Error", fmt.Sprintf("Reached state %v which is not expected",
			CurrentStatus))
		not_deployed_list[serial_number] = true
		return diags, not_deployed_list
	}
	if CurrentStatus == "FAILED" || CurrentStatus == "OUT-OF-SYNC" {
		not_deployed_list[serial_number] = true
		return diags, not_deployed_list
	}

	if expectedStatus == "NA" {
		if CurrentStatus == "NA" {
			not_deployed_list[serial_number] = false
			return diags, not_deployed_list
		}
	} else if expectedStatus == "DEPLOYED" {
		if CurrentStatus == "DEPLOYED" {
			not_deployed_list[serial_number] = false
			return diags, not_deployed_list
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Reached state '%v' which is not expected ", CurrentStatus))
	not_deployed_list[serial_number] = true
	return diags, not_deployed_list
}
