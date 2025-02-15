// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package annotations

import (
	"strings"

	networking "istio.io/api/networking/v1alpha3"

	. "github.com/alibaba/higress/ingress/log"
)

const (
	// request
	requestHeaderAdd    = "request-header-control-add"
	requestHeaderUpdate = "request-header-control-update"
	requestHeaderRemove = "request-header-control-remove"

	// response
	responseHeaderAdd    = "response-header-control-add"
	responseHeaderUpdate = "response-header-control-update"
	responseHeaderRemove = "response-header-control-remove"
)

var (
	_ Parser       = headerControl{}
	_ RouteHandler = headerControl{}
)

type HeaderOperation struct {
	Add    map[string]string
	Update map[string]string
	Remove []string
}

// HeaderControlConfig enforces header operations on route level.
// Note: Canary route don't use header control applied on the normal route.
type HeaderControlConfig struct {
	Request  *HeaderOperation
	Response *HeaderOperation
}

type headerControl struct{}

func (h headerControl) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needHeaderControlConfig(annotations) {
		return nil
	}

	config.HeaderControl = &HeaderControlConfig{}

	var requestAdd map[string]string
	var requestUpdate map[string]string
	var requestRemove []string
	if add, err := annotations.ParseStringForMSE(requestHeaderAdd); err == nil {
		requestAdd = convertAddOrUpdate(add)
	}
	if update, err := annotations.ParseStringForMSE(requestHeaderUpdate); err == nil {
		requestUpdate = convertAddOrUpdate(update)
	}
	if remove, err := annotations.ParseStringForMSE(requestHeaderRemove); err == nil {
		requestRemove = splitBySeparator(remove, ",")
	}
	if len(requestAdd) > 0 || len(requestUpdate) > 0 || len(requestRemove) > 0 {
		config.HeaderControl.Request = &HeaderOperation{
			Add:    requestAdd,
			Update: requestUpdate,
			Remove: requestRemove,
		}
	}

	var responseAdd map[string]string
	var responseUpdate map[string]string
	var responseRemove []string
	if add, err := annotations.ParseStringForMSE(responseHeaderAdd); err == nil {
		responseAdd = convertAddOrUpdate(add)
	}
	if update, err := annotations.ParseStringForMSE(responseHeaderUpdate); err == nil {
		responseUpdate = convertAddOrUpdate(update)
	}
	if remove, err := annotations.ParseStringForMSE(responseHeaderRemove); err == nil {
		responseRemove = splitBySeparator(remove, ",")
	}
	if len(responseAdd) > 0 || len(responseUpdate) > 0 || len(responseRemove) > 0 {
		config.HeaderControl.Response = &HeaderOperation{
			Add:    responseAdd,
			Update: responseUpdate,
			Remove: responseRemove,
		}
	}

	return nil
}

func (h headerControl) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	headerControlConfig := config.HeaderControl
	if headerControlConfig == nil {
		return
	}

	headers := &networking.Headers{
		Request:  &networking.Headers_HeaderOperations{},
		Response: &networking.Headers_HeaderOperations{},
	}
	if headerControlConfig.Request != nil {
		headers.Request.Add = headerControlConfig.Request.Add
		headers.Request.Set = headerControlConfig.Request.Update
		headers.Request.Remove = headerControlConfig.Request.Remove
	}

	if headerControlConfig.Response != nil {
		headers.Response.Add = headerControlConfig.Response.Add
		headers.Response.Set = headerControlConfig.Response.Update
		headers.Response.Remove = headerControlConfig.Response.Remove
	}

	route.Headers = headers
}

func needHeaderControlConfig(annotations Annotations) bool {
	return annotations.HasMSE(requestHeaderAdd) ||
		annotations.HasMSE(requestHeaderUpdate) ||
		annotations.HasMSE(requestHeaderRemove) ||
		annotations.HasMSE(responseHeaderAdd) ||
		annotations.HasMSE(responseHeaderUpdate) ||
		annotations.HasMSE(responseHeaderRemove)
}

func convertAddOrUpdate(headers string) map[string]string {
	result := map[string]string{}
	parts := strings.Split(headers, "\n")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		keyValue := strings.Fields(part)
		if len(keyValue) != 2 {
			IngressLog.Infof("Header format %s is invalid.", keyValue)
			continue
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])
		result[key] = value
	}
	return result
}
