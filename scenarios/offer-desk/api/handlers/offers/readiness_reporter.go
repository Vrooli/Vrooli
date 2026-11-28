package offers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/api-core/discovery"
	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

const readinessPrefix = "OFFER_DESK_DEPLOYMENT_MANAGER_READINESS"

func newReadinessReporter(_ context.Context, criterionID, binding string, getenv func(string) string, resolve func(context.Context) (string, error)) (*readinessreporter.Reporter, error) {
	if getenv == nil {
		return nil, fmt.Errorf("readiness environment reader is not configured")
	}
	token := strings.TrimSpace(getenv(readinessPrefix + "_TOKEN"))
	if token == "" {
		return nil, nil
	}
	policyVersion, err := strconv.ParseInt(strings.TrimSpace(getenv(readinessPrefix+"_POLICY_VERSION")), 10, 32)
	if err != nil || policyVersion <= 0 {
		return nil, fmt.Errorf("readiness policy version must be a positive integer")
	}
	var authorizationEpoch uint64
	if raw := strings.TrimSpace(getenv(readinessPrefix + "_AUTHORIZATION_EPOCH")); raw != "" {
		authorizationEpoch, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("readiness authorization epoch: %w", err)
		}
	}
	return readinessreporter.New(readinessreporter.Config{
		ResolveBaseURL: resolve,
		Token:          token,
		Identity: readinessreporter.Identity{
			Scenario:              strings.TrimSpace(getenv(readinessPrefix + "_SCENARIO")),
			ProfileID:             strings.TrimSpace(getenv(readinessPrefix + "_PROFILE_ID")),
			CandidateCommit:       strings.TrimSpace(getenv(readinessPrefix + "_CANDIDATE_COMMIT")),
			ArtifactDigest:        strings.TrimSpace(getenv(readinessPrefix + "_ARTIFACT_DIGEST")),
			Targets:               splitTargets(getenv(readinessPrefix + "_TARGETS")),
			Channel:               strings.TrimSpace(getenv(readinessPrefix + "_CHANNEL")),
			PolicyVersion:         int32(policyVersion),
			CandidateID:           strings.TrimSpace(getenv(readinessPrefix + "_CANDIDATE_ID")),
			DestinationRevisionID: strings.TrimSpace(getenv(readinessPrefix + "_DESTINATION_REVISION_ID")),
			AuthorizationEpoch:    authorizationEpoch,
		},
		CriterionID: criterionID, ProducerBinding: binding, ProducerVersion: "offer-desk",
	})
}

func resolveDeploymentManagerURL(ctx context.Context, getenv func(string) string) (string, error) {
	if base := strings.TrimSpace(getenv("DEPLOYMENT_MANAGER_API_BASE_URL")); base != "" {
		return base, nil
	}
	return discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
}

func configuredReadinessScenario(getenv func(string) string) string {
	if getenv == nil {
		return ""
	}
	return getenv(readinessPrefix + "_SCENARIO")
}

func splitTargets(raw string) []string {
	var out []string
	for _, target := range strings.Split(raw, ",") {
		if target = strings.TrimSpace(target); target != "" {
			out = append(out, target)
		}
	}
	return out
}
