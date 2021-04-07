/*
 * Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/dcgm"
)

func NewDCGMCollector(c []Counter, useOldNamespace bool) (*DCGMCollector, func(), error) {
	collector := &DCGMCollector{
		Counters:        c,
		DeviceFields:    NewDeviceFields(c),
		UseOldNamespace: useOldNamespace,
	}

	cleanups, err := SetupDcgmFieldsWatch(collector.DeviceFields)
	if err != nil {
		return nil, func() {}, err
	}

	collector.Cleanups = cleanups

	return collector, func() { collector.Cleanup() }, nil
}

func (c *DCGMCollector) Cleanup() {
	for _, c := range c.Cleanups {
		c()
	}
}

func (c *DCGMCollector) GetMetrics() ([][]Metric, error) {
	count, err := dcgm.GetAllDeviceCount()
	if err != nil {
		return nil, err
	}

	metrics := make([][]Metric, count)
	for i := uint(0); i < count; i++ {
		// TODO: This call could be cached
		deviceInfo, err := dcgm.GetDeviceInfo(i)
		if err != nil {
			return nil, err
		}

		vals, err := dcgm.GetLatestValuesForFields(i, c.DeviceFields)
		if err != nil {
			return nil, err
		}

		metrics[i] = ToMetric(vals, c.Counters, deviceInfo, c.UseOldNamespace)
	}

	return metrics, nil
}

func ToMetric(values []dcgm.FieldValue_v1, c []Counter, d dcgm.Device, useOld bool) []Metric {
	var metrics []Metric

	for i, val := range values {
		v := ToString(val)
		// Filter out counters with no value
		if v == SkipDCGMValue {
			continue
		}
		uuid := "UUID"
		if useOld {
			uuid = "uuid"
		}
		m := Metric{
			Name:  c[i].FieldName,
			Value: v,

			UUID:      uuid,
			GPU:       fmt.Sprintf("%d", d.GPU),
			GPUUUID:   d.UUID,
			GPUDevice: fmt.Sprintf("nvidia%d", d.GPU),

			Model:      d.Identifiers.Model,
			Attributes: map[string]string{},
		}
		metrics = append(metrics, m)
	}

	return metrics

}

func ToString(value dcgm.FieldValue_v1) string {
	switch v := value.Int64(); v {
	case dcgm.DCGM_FT_INT32_BLANK:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT32_NOT_FOUND:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT32_NOT_SUPPORTED:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT32_NOT_PERMISSIONED:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT64_BLANK:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT64_NOT_FOUND:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT64_NOT_SUPPORTED:
		return SkipDCGMValue
	case dcgm.DCGM_FT_INT64_NOT_PERMISSIONED:
		return SkipDCGMValue
	}
	switch v := value.Float64(); v {
	case dcgm.DCGM_FT_FP64_BLANK:
		return SkipDCGMValue
	case dcgm.DCGM_FT_FP64_NOT_FOUND:
		return SkipDCGMValue
	case dcgm.DCGM_FT_FP64_NOT_SUPPORTED:
		return SkipDCGMValue
	case dcgm.DCGM_FT_FP64_NOT_PERMISSIONED:
		return SkipDCGMValue
	}
	switch v := value.FieldType; v {
	case dcgm.DCGM_FT_STRING:
		return value.String()
	case dcgm.DCGM_FT_DOUBLE:
		return fmt.Sprintf("%f", value.Float64())
	case dcgm.DCGM_FT_INT64:
		return fmt.Sprintf("%d", value.Int64())
	default:
		return FailedToConvert
	}

	return FailedToConvert
}
