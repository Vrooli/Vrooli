// Responsibility: read Agent Manager's typed Measures procedures for CLI summaries.
package main

import (
	"encoding/json"
	"fmt"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/proto"
)

// RunStatsReport is the `run stats` summary composed from Measures procedures.
// Every field is the provider's own typed response; nothing is recomputed here.
type RunStatsReport struct {
	Status   *measurepb.RunStatusDistributionResponse
	Success  *measurepb.RunSuccessRateResponse
	Duration *measurepb.RunDurationStatisticsResponse
	Cost     *measurepb.RunCostResponse
	Runners  *measurepb.RunnerBreakdownResponse
	Profiles *measurepb.ProfileBreakdownResponse
	Models   *measurepb.ModelBreakdownResponse
	Tools    *measurepb.ToolUsageResponse
	// Raw holds each procedure's wire JSON keyed by measure, for --json.
	Raw map[string]json.RawMessage
}

func (s *RunService) measure(procedure string, request, response proto.Message) ([]byte, error) {
	payload, err := marshalProtoRequest(request)
	if err != nil {
		return nil, err
	}
	body, err := s.api.Request("POST", procedure, nil, payload)
	if err != nil {
		return body, err
	}
	if err := unmarshalProtoResponse(body, response); err != nil {
		return body, fmt.Errorf("decode %s: %w", procedure, err)
	}
	return body, nil
}

// Stats reads the run summary measures for one filter and window.
func (s *RunService) Stats(filter *measurepb.InvocationFilter, window *sharedmeasurepb.TimeWindow) (*RunStatsReport, error) {
	report := &RunStatsReport{
		Status:   &measurepb.RunStatusDistributionResponse{},
		Success:  &measurepb.RunSuccessRateResponse{},
		Duration: &measurepb.RunDurationStatisticsResponse{},
		Cost:     &measurepb.RunCostResponse{},
		Runners:  &measurepb.RunnerBreakdownResponse{},
		Profiles: &measurepb.ProfileBreakdownResponse{},
		Models:   &measurepb.ModelBreakdownResponse{},
		Tools:    &measurepb.ToolUsageResponse{},
		Raw:      map[string]json.RawMessage{},
	}
	reads := []struct {
		key       string
		procedure string
		request   proto.Message
		response  proto.Message
	}{
		{"statusDistribution", measures_v1connect.MeasuresServiceRunStatusDistributionProcedure, &measurepb.RunStatusDistributionRequest{Window: window, Filter: filter}, report.Status},
		{"successRate", measures_v1connect.MeasuresServiceRunSuccessRateProcedure, &measurepb.RunSuccessRateRequest{Window: window, Filter: filter}, report.Success},
		{"duration", measures_v1connect.MeasuresServiceRunDurationStatisticsProcedure, &measurepb.RunDurationStatisticsRequest{Window: window, Filter: filter}, report.Duration},
		{"cost", measures_v1connect.MeasuresServiceRunCostProcedure, &measurepb.RunCostRequest{Window: window, Filter: filter}, report.Cost},
		{"runners", measures_v1connect.MeasuresServiceRunnerBreakdownProcedure, &measurepb.RunnerBreakdownRequest{Window: window, Filter: filter}, report.Runners},
		{"profiles", measures_v1connect.MeasuresServiceProfileBreakdownProcedure, &measurepb.ProfileBreakdownRequest{Window: window, Filter: filter}, report.Profiles},
		{"models", measures_v1connect.MeasuresServiceModelBreakdownProcedure, &measurepb.ModelBreakdownRequest{Window: window, Filter: filter}, report.Models},
		{"tools", measures_v1connect.MeasuresServiceToolUsageProcedure, &measurepb.ToolUsageRequest{Window: window, Filter: filter}, report.Tools},
	}
	for _, read := range reads {
		body, err := s.measure(read.procedure, read.request, read.response)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", read.key, err)
		}
		report.Raw[read.key] = json.RawMessage(body)
	}
	return report, nil
}
