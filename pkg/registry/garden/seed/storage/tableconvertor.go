// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package storage

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/garden"
	gardenhelper "github.com/gardener/gardener/pkg/apis/garden/helper"

	"k8s.io/apimachinery/pkg/api/meta"
	metatable "k8s.io/apimachinery/pkg/api/meta/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

var swaggerMetadataDescriptions = metav1.ObjectMeta{}.SwaggerDoc()

type convertor struct {
	headers []metav1beta1.TableColumnDefinition
}

func newTableConvertor() rest.TableConvertor {
	return &convertor{
		headers: []metav1beta1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name", Description: swaggerMetadataDescriptions["name"]},
			{Name: "Status", Type: "string", Format: "name", Description: swaggerMetadataDescriptions["status"]},
			{Name: "Provider (CloudProfile)", Type: "string", Format: "name", Description: swaggerMetadataDescriptions["provider"]},
			{Name: "Region", Type: "string", Format: "name", Description: swaggerMetadataDescriptions["region"]},
			{Name: "Age", Type: "date", Description: swaggerMetadataDescriptions["creationTimestamp"]},
			{Name: "Version", Type: "string", Format: "name", Description: swaggerMetadataDescriptions["version"]},
			{Name: "K8S Version", Type: "string", Format: "name", Description: swaggerMetadataDescriptions["kubernetesVersion"]},
		},
	}
}

// ConvertToTable converts the output to a table.
func (c *convertor) ConvertToTable(ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var (
		err   error
		table = &metav1beta1.Table{
			ColumnDefinitions: c.headers,
		}
	)

	if m, err := meta.ListAccessor(obj); err == nil {
		table.ResourceVersion = m.GetResourceVersion()
		table.SelfLink = m.GetSelfLink()
		table.Continue = m.GetContinue()
	} else {
		if m, err := meta.CommonAccessor(obj); err == nil {
			table.ResourceVersion = m.GetResourceVersion()
			table.SelfLink = m.GetSelfLink()
		}
	}

	table.Rows, err = metatable.MetaToTableRow(obj, func(obj runtime.Object, m metav1.Object, name, age string) ([]interface{}, error) {
		var (
			seed  = obj.(*garden.Seed)
			cells = []interface{}{}
		)

		gardenletReadyCondition := gardenhelper.GetCondition(seed.Status.Conditions, garden.SeedGardenletReady)
		seedBootstrappedCondition := gardenhelper.GetCondition(seed.Status.Conditions, garden.SeedBootstrapped)

		cells = append(cells, seed.Name)
		if gardenletReadyCondition != nil && gardenletReadyCondition.Status == garden.ConditionUnknown {
			cells = append(cells, "Unknown")
		} else if (gardenletReadyCondition == nil || gardenletReadyCondition.Status != garden.ConditionTrue) ||
			(seedBootstrappedCondition == nil || seedBootstrappedCondition.Status != garden.ConditionTrue) {
			cells = append(cells, "NotReady")
		} else {
			cells = append(cells, "Ready")
		}
		cells = append(cells, fmt.Sprintf("%s (%s)", seed.Spec.Provider.Type, seed.Spec.Cloud.Profile))
		cells = append(cells, seed.Spec.Provider.Region)
		cells = append(cells, metatable.ConvertToHumanReadableDateType(seed.CreationTimestamp))
		if gardener := seed.Status.Gardener; gardener != nil && len(gardener.Version) > 0 {
			cells = append(cells, gardener.Version)
		} else {
			cells = append(cells, "<unknown>")
		}
		if k8sVersion := seed.Status.KubernetesVersion; k8sVersion != nil {
			cells = append(cells, *k8sVersion)
		} else {
			cells = append(cells, "<unknown>")
		}

		return cells, nil
	})

	return table, err
}
