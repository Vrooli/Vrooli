package facts

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const proofAnalyzer = "code-facts.proof"

type endpointDocument struct {
	Endpoints []endpointDeclaration `json:"endpoints"`
}

type endpointDeclaration struct {
	ID            string         `json:"id"`
	Path          string         `json:"path"`
	Method        string         `json:"method"`
	Category      string         `json:"category"`
	RESTException *restException `json:"rest_exception"`
}

type restException struct {
	Reason        string         `json:"reason"`
	Note          string         `json:"note"`
	ProtoPayloads *protoPayloads `json:"proto_payloads"`
}

type protoPayloads struct {
	Request  payloadDeclaration `json:"request"`
	Response payloadDeclaration `json:"response"`
	Error    payloadDeclaration `json:"error"`
}

type payloadDeclaration struct {
	ProtoFullName string `json:"proto_full_name"`
	Transport     string `json:"transport"`
	Conformance   string `json:"conformance"`
}

type proofInput struct {
	target    *factsv1.TargetContext
	facts     []*factsv1.GenericFact
	warnings  []*factsv1.Warning
	evidence  []*factsv1.Evidence
	cache     *factsv1.CacheMetadata
	surfaces  []*factsv1.Surface
	endpoints []endpointDeclaration
}

func (s *Service) proofInput(ctx *factsv1.TargetContext, facts []*factsv1.GenericFact, warnings []*factsv1.Warning, evidence []*factsv1.Evidence, cache *factsv1.CacheMetadata) proofInput {
	return proofInput{
		target:    ctx,
		facts:     facts,
		warnings:  warnings,
		evidence:  evidence,
		cache:     cache,
		surfaces:  discoverSurfaces(ctx),
		endpoints: loadEndpointDeclarations(ctx),
	}
}

func synthesizeProtoAdoption(input proofInput, selected []string) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning) {
	if !input.target.GetScenarioAware() {
		return unsupportedProof(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, "Proto adoption proof requires a Vrooli scenario target.")
	}
	surfaceSet := requestedSet(selected)
	var out []*factsv1.GenericFact
	var evidenceOut []*factsv1.Evidence
	var warnings []*factsv1.Warning
	imports := factsByFamily(input.facts, factsv1.FactFamily_FACT_FAMILY_IMPORTS)
	surfaces := input.surfaces
	sort.SliceStable(surfaces, func(i, j int) bool { return surfaces[i].GetId() < surfaces[j].GetId() })
	for _, surface := range surfaces {
		id := surface.GetId()
		if len(surfaceSet) > 0 && !surfaceSet[id] {
			continue
		}
		if id != "api" && id != "cli" && id != "ui" {
			continue
		}
		matches := matchingProtoImports(input.target.GetScenario(), id, surface.GetPath(), imports)
		status := factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING
		message := "No generated proto imports were found for " + id + "."
		confidence := 0.85
		if surface.GetStatus() == factsv1.SurfaceStatus_SURFACE_STATUS_MISSING {
			status = factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED
			message = id + " surface is absent."
			confidence = 0
		} else if len(matches) > 0 {
			status = factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN
			message = fmt.Sprintf("Found %d generated proto import(s) for %s.", len(matches), id)
			confidence = 1
		}
		attrs := map[string]string{
			"surface":        id,
			"scenario":       input.target.GetScenario(),
			"classification": protoImportClassification(input.target.GetScenario(), matches),
		}
		ev := &factsv1.Evidence{Status: status, Confidence: confidence, Analyzer: proofAnalyzer, Message: message}
		if len(matches) > 0 {
			attrs["import_path"] = matches[0].GetSubject()
			for _, source := range matches[0].GetEvidence() {
				if source.GetRange() != nil {
					ev.Range = source.GetRange()
					break
				}
			}
		} else if surface.GetPath() != "" {
			ev.Range = &factsv1.SourceRange{File: surface.GetPath()}
		}
		fact := &factsv1.GenericFact{
			Id:         "proto_adoption:" + id,
			Family:     factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
			Kind:       "proto_import_adoption",
			Subject:    id,
			Evidence:   []*factsv1.Evidence{ev},
			Attributes: attrs,
		}
		out = append(out, fact)
		evidenceOut = append(evidenceOut, ev)
	}
	return out, evidenceOut, warnings
}

func synthesizeEndpointProofs(input proofInput, selected []string) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning) {
	if !input.target.GetScenarioAware() {
		return unsupportedProof(factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS, "Endpoint proof requires a Vrooli scenario target.")
	}
	selectedSet := requestedSet(selected)
	var out []*factsv1.GenericFact
	var evidenceOut []*factsv1.Evidence
	var warnings []*factsv1.Warning
	for _, endpoint := range input.endpoints {
		if endpoint.RESTException == nil {
			continue
		}
		if len(selectedSet) > 0 && !selectedSet[endpoint.ID] {
			continue
		}
		proofs := endpointPayloadProofs(endpoint, input.facts)
		status := aggregateStatus(proofs)
		message := endpoint.ID + " REST exception proof synthesized from endpoint metadata and graph usage facts."
		ev := &factsv1.Evidence{Status: status, Confidence: confidenceForStatus(status), Analyzer: proofAnalyzer, Message: message, Range: &factsv1.SourceRange{File: endpointsPath(input.target)}}
		out = append(out, &factsv1.GenericFact{
			Id:       "endpoint_proof:" + endpoint.ID,
			Family:   factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
			Kind:     "rest_exception_endpoint_proof",
			Subject:  endpoint.ID,
			Evidence: append([]*factsv1.Evidence{ev}, proofs...),
			Attributes: map[string]string{
				"endpoint_id": endpoint.ID,
				"path":        endpoint.Path,
				"method":      endpoint.Method,
				"reason":      endpoint.RESTException.Reason,
				"status":      status.String(),
			},
		})
		evidenceOut = append(evidenceOut, ev)
		evidenceOut = append(evidenceOut, proofs...)
	}
	if len(out) == 0 {
		status := factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING
		message := "No REST exception endpoints matched the request."
		if len(input.endpoints) == 0 {
			status = factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN
			message = "No endpoint metadata was found for target."
		}
		ev := &factsv1.Evidence{Status: status, Confidence: 0, Analyzer: proofAnalyzer, Message: message, Range: &factsv1.SourceRange{File: endpointsPath(input.target)}}
		out = append(out, &factsv1.GenericFact{
			Id:       "endpoint_proof:none",
			Family:   factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
			Kind:     "rest_exception_endpoint_proof",
			Subject:  "none",
			Evidence: []*factsv1.Evidence{ev},
			Attributes: map[string]string{
				"status": status.String(),
			},
		})
		evidenceOut = append(evidenceOut, ev)
	}
	return out, evidenceOut, warnings
}

func endpointPayloadProofs(endpoint endpointDeclaration, facts []*factsv1.GenericFact) []*factsv1.Evidence {
	payloads := endpoint.RESTException.ProtoPayloads
	if payloads == nil {
		return []*factsv1.Evidence{{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING,
			Confidence: 1,
			Analyzer:   proofAnalyzer,
			Message:    "REST exception has no proto_payloads declaration.",
		}}
	}
	var out []*factsv1.Evidence
	out = append(out, routeEvidence(endpoint, facts))
	out = append(out, payloadEvidence("response", payloads.Response, facts))
	out = append(out, payloadEvidence("error", payloads.Error, facts))
	if payloads.Request.Transport != "" && payloads.Request.Conformance != "none" && payloads.Request.ProtoFullName != "" {
		out = append(out, payloadEvidence("request", payloads.Request, facts))
	}
	return out
}

func routeEvidence(endpoint endpointDeclaration, facts []*factsv1.GenericFact) *factsv1.Evidence {
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS) {
		attrs := fact.GetAttributes()
		if attrs["route_path"] == endpoint.Path && strings.EqualFold(attrs["http_method"], endpoint.Method) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Route registration matched "+endpoint.Method+" "+endpoint.Path+".")
		}
	}
	return &factsv1.Evidence{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
		Confidence: 0,
		Analyzer:   proofAnalyzer,
		Message:    "Route registration could not be proven from current graph facts without explicit route_path/http_method attributes.",
	}
}

func payloadEvidence(role string, payload payloadDeclaration, facts []*factsv1.GenericFact) *factsv1.Evidence {
	if payload.Conformance == "none" || payload.Transport == "none" {
		return &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, Confidence: 0, Analyzer: proofAnalyzer, Message: role + " payload has no proto transport to prove."}
	}
	expected := goPayloadExpectation(payload.ProtoFullName)
	if expected.importPath == "" {
		return &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, Confidence: 0, Analyzer: proofAnalyzer, Message: role + " payload declaration has no proto_full_name."}
	}
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES) {
		attrs := fact.GetAttributes()
		if callUsesPayload(attrs, expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, role+" payload uses "+payload.ProtoFullName+" through a recognized helper or typed argument.")
		}
		if role == "response" && callContradictsPayload(attrs, expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED, "response payload uses a different generated proto type than "+payload.ProtoFullName+".")
		}
	}
	if importPresent(expected.importPath, facts) {
		return &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0.25,
			Analyzer:   proofAnalyzer,
			Message:    role + " payload import is present, but no response/error helper usage was proven for " + payload.ProtoFullName + ".",
		}
	}
	return &factsv1.Evidence{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING,
		Confidence: 0.85,
		Analyzer:   proofAnalyzer,
		Message:    role + " payload " + payload.ProtoFullName + " was not found in imports, references, calls, or type usages.",
	}
}

type goPayload struct {
	importPath string
	typeName   string
	fullName   string
}

func goPayloadExpectation(fullName string) goPayload {
	parts := strings.Split(strings.TrimSpace(fullName), ".")
	if len(parts) < 4 {
		return goPayload{}
	}
	versionIdx := -1
	for i, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) > 1 && part[1] >= '0' && part[1] <= '9' {
			versionIdx = i
			break
		}
	}
	if versionIdx < 2 || versionIdx >= len(parts)-1 {
		return goPayload{}
	}
	scenarioSlug := strings.ReplaceAll(strings.Join(parts[1:versionIdx], "_"), "_", "-")
	pkgParts := parts[versionIdx+1 : len(parts)-1]
	importParts := []string{"github.com/vrooli/vrooli/packages/proto/gen/go", scenarioSlug, parts[versionIdx]}
	importParts = append(importParts, pkgParts...)
	return goPayload{importPath: strings.Join(importParts, "/"), typeName: parts[len(parts)-1], fullName: fullName}
}

func callUsesPayload(attrs map[string]string, expected goPayload) bool {
	haystack := strings.Join([]string{
		attrs["callee"],
		attrs["callee_package"],
		attrs["callee_symbol"],
		attrs["argument_types"],
		attrs["resolved_type"],
		attrs["type"],
	}, " ")
	if !strings.Contains(haystack, expected.typeName) {
		return false
	}
	if strings.Contains(haystack, expected.importPath) {
		return true
	}
	return strings.Contains(haystack, "."+expected.typeName) && strings.Contains(haystack, "WriteProto")
}

func callContradictsPayload(attrs map[string]string, expected goPayload) bool {
	haystack := strings.Join([]string{attrs["callee"], attrs["argument_types"], attrs["resolved_type"], attrs["type"]}, " ")
	if !strings.Contains(haystack, "WriteProto") {
		return false
	}
	return strings.Contains(haystack, "github.com/vrooli/vrooli/packages/proto/gen/go/") &&
		strings.Contains(haystack, "/"+scenarioFromImportPath(expected.importPath)+"/") &&
		!strings.Contains(haystack, expected.importPath+"."+expected.typeName)
}

func proofEvidenceFromFact(fact *factsv1.GenericFact, status factsv1.EvidenceStatus, message string) *factsv1.Evidence {
	ev := &factsv1.Evidence{Status: status, Confidence: confidenceForStatus(status), Analyzer: proofAnalyzer, Symbol: fact.GetSubject(), Message: message}
	for _, source := range fact.GetEvidence() {
		if source.GetRange() != nil {
			ev.Range = source.GetRange()
			break
		}
	}
	return ev
}

func matchingProtoImports(scenario, surfaceID, surfacePath string, facts []*factsv1.GenericFact) []*factsv1.GenericFact {
	var out []*factsv1.GenericFact
	for _, fact := range facts {
		importPath := unquote(firstNonEmpty(fact.GetAttributes()["import_path"], fact.GetSubject()))
		if importPath == "" || !strings.Contains(importPath, "github.com/vrooli/vrooli/packages/proto/gen/") {
			continue
		}
		if surfacePath != "" && factFile(fact) != "" && !isWithinPath(factFile(fact), surfacePath) {
			continue
		}
		switch surfaceID {
		case "api", "cli":
			if strings.Contains(importPath, "/gen/go/"+scenario+"/") || strings.Contains(importPath, "/gen/go/"+scenario+"/v") {
				out = append(out, fact)
			}
		case "ui":
			if strings.Contains(importPath, "/gen/typescript/"+scenario+"/") || strings.Contains(importPath, "/gen/typescript/js/"+scenario+"/") {
				out = append(out, fact)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetId() < out[j].GetId() })
	return out
}

func protoImportClassification(scenario string, matches []*factsv1.GenericFact) string {
	if len(matches) == 0 {
		return "missing"
	}
	importPath := unquote(firstNonEmpty(matches[0].GetAttributes()["import_path"], matches[0].GetSubject()))
	if strings.Contains(importPath, "/"+scenario+"/v") {
		return "scenario_owned"
	}
	if strings.Contains(importPath, "/shared/") || strings.Contains(importPath, "/common/") {
		return "shared"
	}
	return "generated_proto"
}

func importPresent(importPath string, facts []*factsv1.GenericFact) bool {
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_IMPORTS) {
		got := unquote(firstNonEmpty(fact.GetAttributes()["import_path"], fact.GetSubject()))
		if got == importPath {
			return true
		}
	}
	return false
}

func scenarioFromImportPath(importPath string) string {
	parts := strings.Split(importPath, "/")
	for i, part := range parts {
		if part == "go" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func factsByFamily(facts []*factsv1.GenericFact, families ...factsv1.FactFamily) []*factsv1.GenericFact {
	set := map[factsv1.FactFamily]bool{}
	for _, family := range families {
		set[family] = true
	}
	out := make([]*factsv1.GenericFact, 0, len(facts))
	for _, fact := range facts {
		if set[fact.GetFamily()] {
			out = append(out, fact)
		}
	}
	return out
}

func loadEndpointDeclarations(target *factsv1.TargetContext) []endpointDeclaration {
	path := endpointsPath(target)
	var doc endpointDocument
	if readJSON(path, &doc) != nil {
		return nil
	}
	sort.SliceStable(doc.Endpoints, func(i, j int) bool { return doc.Endpoints[i].ID < doc.Endpoints[j].ID })
	return doc.Endpoints
}

func endpointsPath(target *factsv1.TargetContext) string {
	if target == nil || target.GetRootPath() == "" {
		return ""
	}
	return filepath.Join(target.GetRootPath(), ".vrooli", "endpoints.json")
}

func requestedSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func aggregateStatus(evidence []*factsv1.Evidence) factsv1.EvidenceStatus {
	hasUnknown := false
	hasProven := false
	for _, ev := range evidence {
		switch ev.GetStatus() {
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED:
			return factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING:
			return factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
			hasUnknown = true
		case factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN:
			hasProven = true
		}
	}
	if hasUnknown {
		return factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN
	}
	if hasProven {
		return factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN
	}
	return factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED
}

func confidenceForStatus(status factsv1.EvidenceStatus) float64 {
	switch status {
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN:
		return 1
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING, factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED:
		return 0.85
	default:
		return 0
	}
}

func unsupportedProof(family factsv1.FactFamily, message string) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning) {
	ev := &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, Confidence: 0, Analyzer: proofAnalyzer, Message: message}
	return []*factsv1.GenericFact{{
			Id:       strings.ToLower(family.String()) + ".unsupported",
			Family:   family,
			Kind:     "unsupported_proof",
			Subject:  family.String(),
			Evidence: []*factsv1.Evidence{ev},
		}},
		[]*factsv1.Evidence{ev},
		[]*factsv1.Warning{providerWarning(proofAnalyzer, "unsupported", message, factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED)}
}

func factFile(fact *factsv1.GenericFact) string {
	for _, ev := range fact.GetEvidence() {
		if ev.GetRange().GetFile() != "" {
			return ev.GetRange().GetFile()
		}
	}
	return firstNonEmpty(fact.GetAttributes()["file"], fact.GetAttributes()["path"])
}

func isWithinPath(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if unquoted, err := strconvUnquote(value); err == nil {
		return unquoted
	}
	return value
}

func strconvUnquote(value string) (string, error) {
	var decoded string
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return decoded, nil
	}
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") && len(value) >= 2 {
		return strings.Trim(value, "`"), nil
	}
	return "", fmt.Errorf("not quoted")
}
