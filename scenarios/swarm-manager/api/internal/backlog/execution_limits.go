package backlog

import (
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/identity"
)

func executionLimitsFromProto(value *domainpb.ExecutionLimits) *identity.ExecutionLimits {
	if value == nil {
		return nil
	}
	return &identity.ExecutionLimits{MaxSlices: int(value.MaxSlices), MaxTokens: value.MaxTokens, MaxWallSeconds: value.MaxWallSeconds, MaxTurns: int(value.MaxTurns), MaxChargeMicroUSD: value.MaxChargeMicroUsd, MaxChildren: int(value.MaxChildren), MaxNodeAttempts: int(value.MaxNodeAttempts), MaxRetries: int(value.MaxRetries)}
}

func executionLimitsProto(value *identity.ExecutionLimits) *domainpb.ExecutionLimits {
	if value == nil {
		return nil
	}
	return &domainpb.ExecutionLimits{MaxSlices: int32(value.MaxSlices), MaxTokens: value.MaxTokens, MaxWallSeconds: value.MaxWallSeconds, MaxTurns: int32(value.MaxTurns), MaxChargeMicroUsd: value.MaxChargeMicroUSD, MaxChildren: int32(value.MaxChildren), MaxNodeAttempts: int32(value.MaxNodeAttempts), MaxRetries: int32(value.MaxRetries)}
}
