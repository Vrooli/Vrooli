package graph

import (
	"errors"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
)

func ParseNonNegativeIntAttr(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func ScenarioSubdir(scenario, projectPath string) string {
	marker := "/scenarios/" + scenario + "/"
	idx := strings.LastIndex(projectPath, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimSuffix(projectPath[idx+len(marker):], "/")
}

func ClassifyResolveError(err error, scenario string) error {
	kind := "internal"
	var de *discovery.Error
	if errors.As(err, &de) {
		switch de.Kind {
		case discovery.ErrScenarioNotRunning, discovery.ErrVrooliNotFound, discovery.ErrCommandFailed, discovery.ErrInvalidPort:
			kind = "scenario_unreachable"
		case discovery.ErrTimeout:
			kind = "scenario_timeout"
		default:
			kind = "internal"
		}
	}
	return IntegrationError{Kind: kind, Scenario: scenario, Cause: err}
}

func ClassifyConnectError(err error, scenario string) error {
	if err == nil {
		return nil
	}
	kind := "internal"
	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeUnavailable:
			kind = "scenario_unreachable"
		case connect.CodeDeadlineExceeded:
			kind = "scenario_timeout"
		case connect.CodeInvalidArgument:
			kind = "invalid_argument"
		case connect.CodeNotFound:
			kind = "not_found"
		case connect.CodeUnimplemented:
			kind = "unimplemented"
		default:
			kind = "internal"
		}
	}
	return IntegrationError{Kind: kind, Scenario: scenario, Cause: err}
}

func SplitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
