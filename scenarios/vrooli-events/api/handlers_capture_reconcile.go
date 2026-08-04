package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type receiptCaptureDeclaration struct {
	SchemaVersion int `json:"schemaVersion"`
	Policies      []struct {
		PolicyID                string   `json:"policyId"`
		TargetScenario          string   `json:"targetScenario"`
		Operation               string   `json:"operation"`
		Protocol                string   `json:"protocol"`
		ResponseType            string   `json:"responseType"`
		ResponseProjectionPaths []string `json:"responseProjectionPaths"`
		RetentionDays           int      `json:"retentionDays"`
		ReadPrincipals          []string `json:"readPrincipals"`
		NeverExercised          bool     `json:"neverExercised,omitempty"`
	} `json:"policies"`
}

type captureReconcileRequest struct {
	Scenario     string `json:"scenario"`
	DryRun       bool   `json:"dryRun"`
	ValidateOnly bool   `json:"validateOnly"`
}

type scenarioServiceConfig struct {
	Dependencies struct {
		Scenarios map[string]struct {
			Config struct {
				Declarations []string `json:"declarations"`
			} `json:"config"`
		} `json:"scenarios"`
	} `json:"dependencies"`
}

// handleReconcileCapturePolicies applies a scenario's explicitly declared
// receipt policies as an atomic batch. Validation finishes before any write so
// one invalid declaration cannot leave a partially observed policy set.
func (s *Server) handleReconcileCapturePolicies(w http.ResponseWriter, r *http.Request) {
	var request captureReconcileRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Scenario) == "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "scenario is required")
		return
	}
	rules, err := loadCaptureDeclarationRules(request.Scenario)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	if request.DryRun || request.ValidateOnly {
		writeJSON(w, http.StatusOK, map[string]any{"scenario": request.Scenario, "validated": true, "dry_run": request.DryRun, "policies": len(rules)})
		return
	}
	result, err := s.policyStore.ReconcileReceiptProjections(r.Context(), rules)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to reconcile receipt capture policies")
		return
	}
	s.broadcastPolicySnapshot(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"scenario": request.Scenario, "validated": true, "policies": len(rules), "created": result.Created, "updated": result.Updated})
}

func loadCaptureDeclarationRules(scenario string) ([]policy.ReceiptProjectionRule, error) {
	repoRoot := os.Getenv("PROJECT_ROOT")
	if repoRoot == "" {
		repoRoot, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	return loadCaptureDeclarationRulesAtRoot(repoRoot, scenario)
}

func loadCaptureDeclarationRulesAtRoot(repoRoot, scenario string) ([]policy.ReceiptProjectionRule, error) {
	if strings.Contains(scenario, "/") || strings.Contains(scenario, "\\") || scenario == "." || scenario == ".." {
		return nil, fmt.Errorf("scenario must be a scenario slug")
	}
	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario)
	configBytes, err := os.ReadFile(filepath.Join(scenarioRoot, ".vrooli", "service.json"))
	if err != nil {
		return nil, fmt.Errorf("read scenario service declaration: %w", err)
	}
	var config scenarioServiceConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("decode scenario service declaration: %w", err)
	}
	eventsDependency, ok := config.Dependencies.Scenarios["vrooli-events"]
	if !ok || len(eventsDependency.Config.Declarations) == 0 {
		return nil, fmt.Errorf("scenario %q declares no vrooli-events receipt sources", scenario)
	}
	rules := make([]policy.ReceiptProjectionRule, 0)
	seen := map[string]struct{}{}
	for _, source := range eventsDependency.Config.Declarations {
		clean := filepath.ToSlash(filepath.Clean(source))
		if !strings.HasPrefix(clean, ".vrooli/vrooli-events/") || strings.Contains(clean, "../") {
			return nil, fmt.Errorf("receipt declaration source %q must live under .vrooli/vrooli-events/", source)
		}
		bytes, err := os.ReadFile(filepath.Join(scenarioRoot, filepath.FromSlash(clean)))
		if err != nil {
			return nil, fmt.Errorf("read receipt declaration %q: %w", source, err)
		}
		var declaration receiptCaptureDeclaration
		if err := json.Unmarshal(bytes, &declaration); err != nil {
			return nil, fmt.Errorf("decode receipt declaration %q: %w", source, err)
		}
		if declaration.SchemaVersion != 1 {
			return nil, fmt.Errorf("receipt declaration %q must use schemaVersion 1", source)
		}
		for _, entry := range declaration.Policies {
			if _, duplicate := seen[entry.PolicyID]; duplicate {
				return nil, fmt.Errorf("receipt declaration policy_id %q is duplicated", entry.PolicyID)
			}
			seen[entry.PolicyID] = struct{}{}
			candidate := receiptCapturePolicy{PolicyID: entry.PolicyID, Enabled: true, ResponseType: entry.ResponseType, ResponseProjectionPaths: entry.ResponseProjectionPaths, RetentionDays: entry.RetentionDays}
			candidate.Selector.TargetScenario, candidate.Selector.Operation, candidate.Selector.Protocol, candidate.Selector.EventType = entry.TargetScenario, entry.Operation, entry.Protocol, receiptEventType
			candidate.Access.ReadPrincipals = entry.ReadPrincipals
			if message := validateCapturePolicy(candidate); message != "" {
				return nil, fmt.Errorf("receipt declaration %q policy %q: %s", source, entry.PolicyID, message)
			}
			if entry.Protocol == "connect" {
				if err := validateDeclaredResponse(repoRoot, entry.Operation, entry.ResponseType, entry.ResponseProjectionPaths); err != nil {
					return nil, fmt.Errorf("receipt declaration %q policy %q: %w", source, entry.PolicyID, err)
				}
			} else if !validHTTPCaptureOperation(entry.Operation) {
				return nil, fmt.Errorf("receipt declaration %q policy %q: HTTP operation must be a stable POST path", source, entry.PolicyID)
			}
			rule := candidate.rule()
			rule.NeverExercised = entry.NeverExercised
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func validHTTPCaptureOperation(operation string) bool {
	parts := strings.Fields(operation)
	return len(parts) == 2 && parts[0] == http.MethodPost && strings.HasPrefix(parts[1], "/") && !strings.Contains(parts[1], "?")
}

// validateDeclaredResponse binds a capture declaration to the target's published
// Connect descriptor. Declarations are consumer-authored, but may only project
// fields explicitly present on the operation's actual response message.
func validateDeclaredResponse(repoRoot, operation, responseType string, paths []string) error {
	parts := strings.Fields(operation)
	if len(parts) != 2 || parts[0] != http.MethodPost {
		return fmt.Errorf("operation %q must be a POST Connect operation", operation)
	}
	operationParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(operationParts) != 2 || operationParts[0] == "" || operationParts[1] == "" {
		return fmt.Errorf("operation %q must name a Connect service and method", operation)
	}
	descriptorBytes, err := os.ReadFile(filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"))
	if err != nil {
		return fmt.Errorf("read protobuf descriptor image: %w", err)
	}
	var image descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(descriptorBytes, &image); err != nil {
		return fmt.Errorf("decode protobuf descriptor image: %w", err)
	}

	var outputType string
	messages := make(map[string]*descriptorpb.DescriptorProto)
	for _, file := range image.File {
		pkg := file.GetPackage()
		for _, message := range file.MessageType {
			indexDescriptorMessage(messages, pkg, "", message)
		}
		for _, service := range file.Service {
			if joinProtoName(pkg, service.GetName()) != operationParts[0] {
				continue
			}
			for _, method := range service.Method {
				if method.GetName() == operationParts[1] {
					outputType = strings.TrimPrefix(method.GetOutputType(), ".")
					break
				}
			}
		}
	}
	if outputType == "" {
		return fmt.Errorf("operation %q is not in the protobuf descriptor image", operation)
	}
	if responseType != outputType {
		return fmt.Errorf("response_type %q does not match operation response %q", responseType, outputType)
	}
	message := messages[outputType]
	if message == nil {
		return fmt.Errorf("response_type %q is not a protobuf message", responseType)
	}
	for _, path := range paths {
		if err := validateProjectionPath(messages, message, path); err != nil {
			return fmt.Errorf("response projection %q: %w", path, err)
		}
	}
	return nil
}

func indexDescriptorMessage(messages map[string]*descriptorpb.DescriptorProto, pkg, parent string, message *descriptorpb.DescriptorProto) {
	name := joinProtoName(pkg, message.GetName())
	if parent != "" {
		name = parent + "." + message.GetName()
	}
	messages[name] = message
	for _, nested := range message.NestedType {
		indexDescriptorMessage(messages, pkg, name, nested)
	}
}

func joinProtoName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func validateProjectionPath(messages map[string]*descriptorpb.DescriptorProto, message *descriptorpb.DescriptorProto, path string) error {
	segments := strings.Split(path, ".")
	for index, segment := range segments {
		var field *descriptorpb.FieldDescriptorProto
		for _, candidate := range message.Field {
			if candidate.GetName() == segment {
				field = candidate
				break
			}
		}
		if field == nil {
			return fmt.Errorf("field %q does not exist on %q", segment, message.GetName())
		}
		if index == len(segments)-1 {
			return nil
		}
		if field.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE && field.GetType() != descriptorpb.FieldDescriptorProto_TYPE_GROUP {
			return fmt.Errorf("field %q is not a message", segment)
		}
		message = messages[strings.TrimPrefix(field.GetTypeName(), ".")]
		if message == nil {
			return fmt.Errorf("field %q has an unresolved message type", segment)
		}
	}
	return nil
}
