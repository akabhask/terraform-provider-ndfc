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

package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-nd"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	NDFC_CHECK_STATUS_RETRIES = 3
	NDFC_CHECK_STATUS_DELAY   = 4
)

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func GetListString(result []gjson.Result) types.List {
	v := make([]attr.Value, len(result))
	for r := range result {
		v[r] = types.StringValue(result[r].String())
	}
	return types.ListValueMust(types.StringType, v)
}

func DeployInterface(ctx context.Context, client *nd.Client, serialNumber, interfaceName string) diag.Diagnostics {
	var diags diag.Diagnostics
	id := serialNumber + "/" + interfaceName
	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Deploy Interface", id))

	body := ""
	body, _ = sjson.Set(body, "0.serialNumber", serialNumber)
	body, _ = sjson.Set(body, "0.ifName", interfaceName)
	res, err := client.Post("/lan-fabric/rest/interface/deploy", body)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Failed to deploy interface, got error: %s, %s", err, res.String()))
		return diags
	}

	t := res.Get("0.reportItemType").String()
	if t == "ERROR" {
		diags.AddError("Client Error", fmt.Sprintf("Failed to deploy interface (%s, %s), got error: %s", serialNumber, interfaceName, res.Get("0.message").String()))
		return diags
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Deploy Interface finished successfully", id))

	return diags
}

func CheckAttachmentResponse(ctx context.Context, response gjson.Result) diag.Diagnostics {
	var diags diag.Diagnostics
	response.ForEach(func(k, v gjson.Result) bool {
		if !strings.Contains(v.String(), "SUCCESS") && !strings.Contains(v.String(), "already in detached state") {
			diags.AddError("Client Error", fmt.Sprintf("Failed to configure attachments, got error: %s, %s", k.String(), v.String()))
		}
		return true
	})
	return diags
}
