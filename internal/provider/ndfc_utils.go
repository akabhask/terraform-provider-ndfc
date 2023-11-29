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
type ndfcGetPathName interface {
	getPath()
}
type ndfcRestApiRequest interface {
	ndfcRestApiRequest()
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
		if strings.Contains(res.String(), "already exists") {
			// if entity already exists, continue with operation
			tflog.Debug(ctx, fmt.Sprintf("already exists continue with next operation : payload : %v", payLoad))
			return res, nil, diags
		}
		if strings.Contains(res.String(), "connect: operation timed out") {
			// Sometimes connection to NDFC fails, we retry till we connect
			tflog.Debug(ctx, "operation timed out")
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

func (client *NdfcClient) WaitForStatus(ctx context.Context, state VRF, expectedStatus string) bool {
	for i := 0; i < (helpers.FABRIC_DEPLOY_TIMEOUT / 5); i++ {
		time.Sleep(1 * time.Second)
		res, err, _ := client.ndfcRestApiRequest(ctx, "GET",
			fmt.Sprintf("%v", state.getPath()), "")
		if err != nil {
			continue
		}
		status := res.Get(`#(vrfName="` + state.VrfName.ValueString() + `").vrfStatus`).String()
		if strings.Contains(status, expectedStatus) {
			return client.checkStateStabilized(ctx, state, expectedStatus, 10)
		}
	}
	return failed
}
func (r *NdfcClient) checkStateStabilized(ctx context.Context, state VRF, expectedStatus string, retry int) bool {
	for i := 0; i < retry; i++ {
		time.Sleep(3 * time.Second)
		res, err, _ := r.ndfcRestApiRequest(ctx, "GET", state.getPath(), "")
		if err != nil {
			continue
		}
		status := res.Get(`#(vrfName="` + state.VrfName.ValueString() + `").vrfStatus`).String()
		tflog.Debug(ctx, fmt.Sprintf(" checkExpectedState status: %v %v %v", status, i, retry))
		if !strings.Contains(status, expectedStatus) {
			return failed
		}
	}
	return success
}

func (client *NdfcClient) Deploy(ctx context.Context, state VRF, expectedStatus string) diag.Diagnostics {
	var diags diag.Diagnostics
	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Deploy", state.Id.ValueString()))

	body := ""
	body, _ = sjson.Set(body, "vrfNames", state.VrfName.ValueString())
	res, err, diags := client.ndfcRestApiRequest(ctx, "GET", state.getPath(), "")
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Failed to retrieve VRFs, got error: %s, %s", err, res.String()))
		return diags
	}
	vrfStatus := res.Get(`#(vrfName="` + state.VrfName.ValueString() + `").vrfStatus`).String()
	if expectedStatus == vrfStatus {
		stateReached := client.checkStateStabilized(ctx, state, expectedStatus, 1)
		log.Printf("stateReached: %v", stateReached)
		if stateReached == success {
			return diags
		}
	}
	switch vrfStatus {
	case "IN PROGRESS":
	case "DEPLOYED":
	case "OUT-OF-SYNC":
		fallthrough
	case "PENDING":
		fallthrough
	case "NA":
		res, err, diags = client.ndfcRestApiRequest(ctx, "POST", state.getPath()+"deployments", body)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Failed to POST, got error: %s, %s", err, res.String()))
			return diags
		}
	default:
		diags.AddError("Client Error", fmt.Sprintf("Invalid state reached in Deploy: %s, %s", err, res.String()))
		return diags
	}
	stateReached := client.WaitForStatus(ctx, state, expectedStatus)
	if stateReached == failed {
		time.Sleep(5 * time.Second)
		res, err, diags = client.ndfcRestApiRequest(ctx, "POST", state.getPath()+"deployments", body)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Failed to POST, got error: %s, %s", err, res.String()))
			return diags
		}
	}
	return diags
}
