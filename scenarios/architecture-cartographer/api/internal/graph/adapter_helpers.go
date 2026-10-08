package graph

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type ScenarioURLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

type ProjectPathFn func(scenarioName string) (path string, found bool, err error)

func ResolveAdapterTarget(
	ctx context.Context,
	urls ScenarioURLResolver,
	projectOf ProjectPathFn,
	targetScenario string,
	producerScenario string,
	languageName string,
) (projectPath string, baseURL string, found bool, err error) {
	if projectOf == nil || urls == nil {
		return "", "", false, IntegrationError{
			Kind:     "internal",
			Scenario: producerScenario,
			Cause:    fmt.Errorf("%s adapter not fully configured (missing URLResolver or ProjectPath)", languageName),
		}
	}
	projectPath, found, err = projectOf(targetScenario)
	if err != nil {
		return "", "", false, IntegrationError{
			Kind:     "internal",
			Scenario: producerScenario,
			Cause:    fmt.Errorf("resolve %s project path for %q: %w", languageName, targetScenario, err),
		}
	}
	if !found {
		return projectPath, "", false, nil
	}
	baseURL, err = urls.ResolveScenarioURLDefault(ctx, producerScenario)
	if err != nil {
		return "", "", false, ClassifyResolveError(err, producerScenario)
	}
	return projectPath, baseURL, true, nil
}

func ExtractFromProject(
	ctx context.Context,
	urls ScenarioURLResolver,
	projectOf ProjectPathFn,
	targetScenario string,
	producerScenario string,
	languageName string,
	call func(context.Context, string, string) (RawGraph, error),
) (RawGraph, error) {
	projectPath, baseURL, found, err := ResolveAdapterTarget(ctx, urls, projectOf, targetScenario, producerScenario, languageName)
	if err != nil {
		return RawGraph{}, err
	}
	if !found {
		return RawGraph{}, nil
	}
	raw, err := call(ctx, baseURL, projectPath)
	if err != nil {
		return RawGraph{}, err
	}
	RebaseFilesToScenario(&raw, targetScenario, projectPath)
	return raw, nil
}

func RebaseFilesToScenario(raw *RawGraph, scenario, projectPath string) {
	if raw == nil {
		return
	}
	if subdir := ScenarioSubdir(scenario, projectPath); subdir != "" {
		for i := range raw.Files {
			raw.Files[i].Path = subdir + "/" + raw.Files[i].Path
		}
	}
	AssignPackageRepoPaths(raw.Packages, raw.Files)
}

func FileNodeFromProto(n *commonv1.CodeGraphNode, language Language) FileNode {
	attrs := n.GetAttributes()
	return FileNode{
		ID:        n.GetId(),
		Path:      n.GetPath(),
		PackageID: attrs["package_id"],
		Language:  language,
		Lines:     ParseNonNegativeIntAttr(attrs["lines"]),
		IsTest:    attrs["is_test"] == "true",
	}
}

func ImportEdgeFromProto(e *commonv1.CodeGraphEdge) ImportEdge {
	attrs := e.GetAttributes()
	return ImportEdge{
		From:        e.GetFromNodeId(),
		ToPackageID: e.GetToNodeId(),
		SymbolIDs:   SplitCSV(attrs["symbol_ids"]),
		SymbolKinds: SplitCSV(attrs["symbol_kinds"]),
		TestOnly:    attrs["test_only"] == "true",
	}
}

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
