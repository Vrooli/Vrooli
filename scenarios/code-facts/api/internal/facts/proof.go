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

type endpointProofScope struct {
	route     *factsv1.Evidence
	proven    bool
	parseUnit string
	fileID    string
	file      string
	framework string
	handler   string
	symbol    string
	enclosing string
	factories []string
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
	implementations := defaultEndpointAdapters().ExtractEndpointImplementations(endpointAdapterContext{
		target:    input.target,
		facts:     input.facts,
		endpoints: input.endpoints,
	})
	implementationsByEndpoint := map[string][]endpointImplementation{}
	for _, impl := range implementations {
		implementationsByEndpoint[impl.EndpointID] = append(implementationsByEndpoint[impl.EndpointID], impl)
	}
	var out []*factsv1.GenericFact
	var evidenceOut []*factsv1.Evidence
	warnings := unsupportedEndpointFrameworkWarnings(input.facts)
	for _, endpoint := range input.endpoints {
		if endpoint.RESTException == nil {
			continue
		}
		if len(selectedSet) > 0 && !selectedSet[endpoint.ID] {
			continue
		}
		proofs := endpointImplementationProofs(endpoint, implementationsByEndpoint[endpoint.ID])
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

func unsupportedEndpointFrameworkWarnings(facts []*factsv1.GenericFact) []*factsv1.Warning {
	supported := map[string]bool{
		"go:":                true,
		"go:go.http":         true,
		"go:gorilla/mux":     true,
		"go:net/http":        true,
		"typescript:express": true,
	}
	seen := map[string]bool{}
	var warnings []*factsv1.Warning
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS) {
		attrs := fact.GetAttributes()
		if !isRouteRegistrationFact(attrs) {
			continue
		}
		language := strings.TrimSpace(attrs["language"])
		framework := strings.TrimSpace(attrs["router_framework"])
		if language == "" {
			continue
		}
		key := language + ":" + framework
		if supported[key] || seen[key] {
			continue
		}
		seen[key] = true
		label := framework
		if label == "" {
			label = "unspecified"
		}
		warnings = append(warnings, providerWarning(
			"code-facts.endpoint_proof",
			"framework_unsupported",
			"Endpoint proof has no adapter for "+language+" "+label+" route registrations.",
			factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		))
	}
	sort.SliceStable(warnings, func(i, j int) bool { return warnings[i].GetMessage() < warnings[j].GetMessage() })
	return warnings
}

func isRouteRegistrationFact(attrs map[string]string) bool {
	kind := strings.ToLower(strings.TrimSpace(attrs["kind"]))
	return strings.Contains(kind, "route_registration") ||
		(strings.TrimSpace(attrs["route_path"]) != "" && strings.TrimSpace(attrs["http_method"]) != "")
}

func endpointImplementationProofs(endpoint endpointDeclaration, implementations []endpointImplementation) []*factsv1.Evidence {
	payloads := endpoint.RESTException.ProtoPayloads
	if payloads == nil {
		return []*factsv1.Evidence{{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING,
			Confidence: 1,
			Analyzer:   proofAnalyzer,
			Message:    "REST exception has no proto_payloads declaration.",
		}}
	}
	if len(implementations) == 0 {
		return endpointProofsWithoutImplementation(endpoint)
	}
	impl := bestEndpointImplementation(implementations)
	var out []*factsv1.Evidence
	out = append(out, impl.Evidence...)
	return out
}

func bestEndpointImplementation(implementations []endpointImplementation) endpointImplementation {
	best := implementations[0]
	bestRank := endpointImplementationRank(best)
	for _, impl := range implementations[1:] {
		rank := endpointImplementationRank(impl)
		if rank > bestRank {
			best = impl
			bestRank = rank
		}
	}
	return best
}

func endpointImplementationRank(impl endpointImplementation) int {
	switch aggregateStatus(impl.Evidence) {
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED:
		return 5
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN:
		return 4
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING:
		return 3
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN:
		return 2
	case factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED:
		return 1
	default:
		return 0
	}
}

func endpointProofsWithoutImplementation(endpoint endpointDeclaration) []*factsv1.Evidence {
	payloads := endpoint.RESTException.ProtoPayloads
	out := []*factsv1.Evidence{{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
		Confidence: 0,
		Analyzer:   proofAnalyzer,
		Message:    "Route registration could not be proven by any endpoint implementation adapter.",
	}}
	for _, payload := range declaredPayloadRoles(endpoint) {
		out = append(out, &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    string(payload.role) + " payload proof requires a proven route registration before handler-local payload evidence can be trusted.",
		})
	}
	if payloads != nil && len(out) == 1 {
		out = append(out, &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    "REST exception declares no required proto payload roles.",
		})
	}
	return out
}

func routeProofScope(endpoint endpointDeclaration, facts []*factsv1.GenericFact) endpointProofScope {
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS) {
		attrs := fact.GetAttributes()
		if attrs["route_path"] == endpoint.Path && strings.EqualFold(attrs["http_method"], endpoint.Method) {
			return endpointProofScope{
				route:     proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Route registration matched "+endpoint.Method+" "+endpoint.Path+"."),
				proven:    true,
				parseUnit: attrs["parse_unit_id"],
				fileID:    attrs["file_id"],
				file:      factFile(fact),
				framework: strings.TrimSpace(attrs["router_framework"]),
				handler:   strings.TrimSpace(attrs["handler_expr"]),
				symbol:    strings.TrimSpace(attrs["handler_symbol"]),
				enclosing: strings.TrimSpace(attrs["enclosing_symbol"]),
				factories: handlerFactories(fact, facts),
			}
		}
	}
	return endpointProofScope{
		route: &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    "Route registration could not be proven from current graph facts without explicit route_path/http_method attributes.",
		},
	}
}

func tsExpressRouteProofScope(endpoint endpointDeclaration, facts []*factsv1.GenericFact) endpointProofScope {
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS) {
		attrs := fact.GetAttributes()
		if attrs["language"] != "typescript" || attrs["router_framework"] != "express" {
			continue
		}
		if attrs["route_path_status"] != "proven" || attrs["route_path"] != endpoint.Path || !strings.EqualFold(attrs["http_method"], endpoint.Method) {
			continue
		}
		return endpointProofScope{
			route:     proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Express route registration matched "+endpoint.Method+" "+endpoint.Path+"."),
			proven:    true,
			parseUnit: attrs["parse_unit_id"],
			fileID:    attrs["file_id"],
			file:      factFile(fact),
			framework: "express",
			handler:   strings.TrimSpace(attrs["handler_expr"]),
			symbol:    strings.TrimSpace(attrs["handler_symbol"]),
			enclosing: strings.TrimSpace(firstNonEmpty(attrs["enclosing_symbol"], attrs["enclosing_declaration"])),
		}
	}
	return endpointProofScope{
		route: &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    "Express route registration could not be proven from current TypeScript graph facts with literal route_path/http_method attributes.",
		},
		framework: "express",
	}
}

func goPayloadEvidence(role string, payload payloadDeclaration, facts []*factsv1.GenericFact, scope endpointProofScope) *factsv1.Evidence {
	if payload.Conformance == "none" || payload.Conformance == "transport_only" || payload.Conformance == "external_shape" || payload.Transport == "none" {
		return &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, Confidence: 0, Analyzer: proofAnalyzer, Message: role + " payload has no proto transport to prove."}
	}
	expected := goPayloadExpectation(payload.ProtoFullName)
	if expected.importPath == "" {
		return &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, Confidence: 0, Analyzer: proofAnalyzer, Message: role + " payload declaration has no proto_full_name."}
	}
	if !scope.proven {
		return &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    role + " payload proof requires a proven route registration before handler-local payload evidence can be trusted.",
		}
	}
	scoped := factsInEndpointScope(facts, scope)
	if len(scoped) == 0 {
		return &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    role + " payload proof could not correlate the route registration to a handler scope.",
		}
	}
	for _, fact := range factsByFamily(scoped, factsv1.FactFamily_FACT_FAMILY_CALLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES) {
		attrs := fact.GetAttributes()
		if callUsesPayload(attrs, expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, role+" payload uses "+payload.ProtoFullName+" through a recognized helper or typed argument.")
		}
		if role == "error" && callUsesErrorHelper(attrs) && payloadUsagePresent(factsInParseUnit(facts, scope.parseUnit), expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, role+" payload uses "+payload.ProtoFullName+" through a route-local error helper.")
		}
		if role == "response" && callContradictsPayload(attrs, expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED, "response payload uses a different generated proto type than "+payload.ProtoFullName+".")
		}
	}
	if importPresent(expected.importPath, factsInParseUnit(facts, scope.parseUnit)) {
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

func tsExpressPayloadEvidence(role string, payload payloadDeclaration, facts []*factsv1.GenericFact, scope endpointProofScope) *factsv1.Evidence {
	if payload.Conformance == "none" || payload.Conformance == "transport_only" || payload.Conformance == "external_shape" || payload.Transport == "none" {
		return &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, Confidence: 0, Analyzer: proofAnalyzer, Message: role + " payload has no proto transport to prove."}
	}
	expected := tsPayloadExpectation(payload.ProtoFullName)
	if expected.importPath == "" {
		return &factsv1.Evidence{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, Confidence: 0, Analyzer: proofAnalyzer, Message: role + " payload declaration has no proto_full_name."}
	}
	if !scope.proven {
		return &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    role + " payload proof requires a proven route registration before handler-local payload evidence can be trusted.",
		}
	}
	scoped := factsInEndpointScope(facts, scope)
	if len(scoped) == 0 {
		return &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0,
			Analyzer:   proofAnalyzer,
			Message:    role + " payload proof could not correlate the Express route registration to a handler scope.",
		}
	}
	for _, fact := range factsByFamily(scoped, factsv1.FactFamily_FACT_FAMILY_CALLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES) {
		attrs := fact.GetAttributes()
		if tsCallUsesPayload(attrs, expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, role+" payload uses "+payload.ProtoFullName+" through a recognized Express response call or typed argument.")
		}
		if role == "response" && tsCallContradictsPayload(attrs, expected) {
			return proofEvidenceFromFact(fact, factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED, "response payload uses a different generated TypeScript proto type than "+payload.ProtoFullName+".")
		}
	}
	if tsImportPresent(expected.importPath, factsInParseUnit(facts, scope.parseUnit)) {
		return &factsv1.Evidence{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
			Confidence: 0.25,
			Analyzer:   proofAnalyzer,
			Message:    role + " payload import is present, but no Express response/error usage was proven for " + payload.ProtoFullName + ".",
		}
	}
	return &factsv1.Evidence{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING,
		Confidence: 0.85,
		Analyzer:   proofAnalyzer,
		Message:    role + " payload " + payload.ProtoFullName + " was not found in imports, references, calls, or type usages.",
	}
}

func factsInEndpointScope(facts []*factsv1.GenericFact, scope endpointProofScope) []*factsv1.GenericFact {
	var out []*factsv1.GenericFact
	for _, fact := range facts {
		if fact == nil || !sameParseUnit(fact, scope.parseUnit) {
			continue
		}
		attrs := fact.GetAttributes()
		if endpointScopeMatches(attrs, factFile(fact), scope) {
			out = append(out, fact)
		}
	}
	return out
}

func endpointScopeMatches(attrs map[string]string, file string, scope endpointProofScope) bool {
	if scope.symbol != "" && attrs["enclosing_symbol"] == scope.symbol {
		return true
	}
	enclosing := strings.TrimSpace(firstNonEmpty(attrs["enclosing_symbol"], attrs["enclosing_declaration"]))
	if enclosing != "" && handlerNamesMatch(scope.handler, enclosing) {
		return true
	}
	if scope.fileID != "" && attrs["file_id"] == scope.fileID && enclosing != "" && enclosing == scope.enclosing && handlerNamesMatch(scope.handler, enclosing) {
		return true
	}
	if scope.file != "" && file != "" && cleanEvidencePath(file) == cleanEvidencePath(scope.file) && enclosing != "" && handlerNamesMatch(scope.handler, enclosing) {
		return true
	}
	for _, factory := range scope.factories {
		if handlerNamesMatch(factory, enclosing) {
			return true
		}
	}
	return false
}

func handlerFactories(route *factsv1.GenericFact, facts []*factsv1.GenericFact) []string {
	routeAttrs := route.GetAttributes()
	handlerExpr := strings.TrimSpace(routeAttrs["handler_expr"])
	if handlerExpr == "" || strings.Contains(handlerExpr, ".") {
		return nil
	}
	routeLine := atoiString(routeAttrs["start_line"])
	var out []string
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS) {
		if fact == nil || fact.GetId() == route.GetId() || !sameParseUnit(fact, routeAttrs["parse_unit_id"]) {
			continue
		}
		attrs := fact.GetAttributes()
		if routeAttrs["file_id"] == "" || attrs["file_id"] != routeAttrs["file_id"] {
			continue
		}
		if routeAttrs["enclosing_symbol"] == "" || attrs["enclosing_symbol"] != routeAttrs["enclosing_symbol"] {
			continue
		}
		if routeLine > 0 {
			line := atoiString(attrs["start_line"])
			if line > 0 && line > routeLine {
				continue
			}
		}
		callee := strings.TrimSpace(firstNonEmpty(attrs["callee"], attrs["name"]))
		if callee == "" || !strings.Contains(callee, "Handler") {
			continue
		}
		out = append(out, callee)
	}
	sort.Strings(out)
	return out
}

func handlerNamesMatch(handlerExpr, enclosing string) bool {
	handlerExpr = strings.TrimSpace(handlerExpr)
	enclosing = strings.TrimSpace(enclosing)
	if handlerExpr == "" || enclosing == "" {
		return false
	}
	if handlerExpr == enclosing || strings.HasSuffix(handlerExpr, "."+enclosing) {
		return true
	}
	if strings.HasSuffix(handlerExpr, "("+enclosing+")") {
		return true
	}
	handlerName := handlerLeafName(handlerExpr)
	enclosingName := handlerLeafName(enclosing)
	return handlerName != "" && handlerName == enclosingName
}

func handlerLeafName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "*")
	if idx := strings.LastIndex(value, "."); idx >= 0 {
		value = value[idx+1:]
	}
	value = strings.TrimPrefix(value, "*")
	if idx := strings.LastIndex(value, ")"); idx >= 0 && idx+1 < len(value) {
		value = value[idx+1:]
	}
	return value
}

func atoiString(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func factsInParseUnit(facts []*factsv1.GenericFact, parseUnit string) []*factsv1.GenericFact {
	if parseUnit == "" {
		return facts
	}
	var out []*factsv1.GenericFact
	for _, fact := range facts {
		if sameParseUnit(fact, parseUnit) {
			out = append(out, fact)
		}
	}
	return out
}

func sameParseUnit(fact *factsv1.GenericFact, parseUnit string) bool {
	return parseUnit == "" || fact.GetAttributes()["parse_unit_id"] == parseUnit
}

type goPayload struct {
	importPath string
	typeName   string
	fullName   string
}

type tsPayload struct {
	importPath string
	typeName   string
	fullName   string
	scenario   string
}

func tsPayloadExpectation(fullName string) tsPayload {
	parts := strings.Split(strings.TrimSpace(fullName), ".")
	if len(parts) < 4 {
		return tsPayload{}
	}
	versionIdx := -1
	for i, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) > 1 && part[1] >= '0' && part[1] <= '9' {
			versionIdx = i
			break
		}
	}
	if versionIdx < 2 || versionIdx >= len(parts)-1 {
		return tsPayload{}
	}
	scenarioSlug := strings.ReplaceAll(strings.Join(parts[1:versionIdx], "_"), "_", "-")
	pkgParts := parts[versionIdx+1 : len(parts)-1]
	importParts := []string{"@vrooli/proto-types", scenarioSlug, parts[versionIdx]}
	importParts = append(importParts, pkgParts...)
	return tsPayload{importPath: strings.Join(importParts, "/"), typeName: parts[len(parts)-1], fullName: fullName, scenario: scenarioSlug}
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

func tsCallUsesPayload(attrs map[string]string, expected tsPayload) bool {
	callee := attrs["callee"]
	if !strings.Contains(callee, "res.json") && !strings.Contains(callee, "reply.send") {
		return false
	}
	haystack := strings.Join([]string{
		callee,
		attrs["argument_summary"],
		attrs["argument_types"],
		attrs["resolved_type"],
		attrs["return_type"],
		attrs["type"],
	}, " ")
	return strings.Contains(haystack, expected.importPath+"."+expected.typeName) ||
		strings.Contains(haystack, expected.importPath+"/"+expected.typeName)
}

func tsCallContradictsPayload(attrs map[string]string, expected tsPayload) bool {
	haystack := strings.Join([]string{attrs["callee"], attrs["argument_summary"], attrs["argument_types"], attrs["resolved_type"], attrs["return_type"], attrs["type"]}, " ")
	if !strings.Contains(haystack, "res.json") && !strings.Contains(haystack, "reply.send") {
		return false
	}
	return strings.Contains(haystack, "@vrooli/proto-types/"+expected.scenario+"/") &&
		!strings.Contains(haystack, expected.importPath+"."+expected.typeName) &&
		!strings.Contains(haystack, expected.importPath+"/"+expected.typeName)
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

func tsImportPresent(importPath string, facts []*factsv1.GenericFact) bool {
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_IMPORTS) {
		normalized := normalizedImportPath(fact)
		if normalized == importPath || strings.HasPrefix(normalized, importPath+"/") {
			return true
		}
	}
	return false
}

func callUsesErrorHelper(attrs map[string]string) bool {
	callee := strings.TrimSpace(attrs["callee"])
	return callee == "httpx.WriteError" || strings.HasSuffix(callee, ".WriteError") || callee == "WriteError"
}

func payloadUsagePresent(facts []*factsv1.GenericFact, expected goPayload) bool {
	for _, fact := range factsByFamily(facts, factsv1.FactFamily_FACT_FAMILY_CALLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES) {
		if callUsesPayload(fact.GetAttributes(), expected) {
			return true
		}
	}
	return false
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
		importPath := normalizedImportPath(fact)
		if importPath == "" || !strings.Contains(importPath, "github.com/vrooli/vrooli/packages/proto/gen/") {
			if !strings.Contains(importPath, "@vrooli/proto-types/") {
				continue
			}
		}
		if surfacePath != "" && factFile(fact) != "" && !isWithinPath(factFile(fact), surfacePath) {
			continue
		}
		if surfacePath != "" && !factBelongsToSurface(fact, surfacePath) {
			continue
		}
		switch surfaceID {
		case "api", "cli":
			if strings.Contains(importPath, "/gen/go/"+scenario+"/") || strings.Contains(importPath, "/gen/go/"+scenario+"/v") {
				out = append(out, fact)
			}
		case "ui":
			if strings.Contains(importPath, "/gen/typescript/"+scenario+"/") ||
				strings.Contains(importPath, "/gen/typescript/js/"+scenario+"/") ||
				strings.Contains(importPath, "@vrooli/proto-types/"+scenario+"/") {
				out = append(out, fact)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetId() < out[j].GetId() })
	return out
}

func factBelongsToSurface(fact *factsv1.GenericFact, surfacePath string) bool {
	parseRoot := parseUnitRoot(fact.GetAttributes()["parse_unit_id"])
	if parseRoot != "" {
		rel, err := filepath.Rel(filepath.Clean(surfacePath), filepath.Clean(parseRoot))
		return err == nil && (rel == "." || !strings.HasPrefix(rel, ".."))
	}
	file := factFile(fact)
	if file == "" {
		return false
	}
	if filepath.IsAbs(file) {
		return isWithinPath(file, surfacePath)
	}
	return strings.HasPrefix(cleanEvidencePath(file), cleanEvidencePath(filepath.Base(surfacePath))+"/")
}

func parseUnitRoot(id string) string {
	_, root, ok := strings.Cut(id, ":")
	if !ok {
		return ""
	}
	return root
}

func protoImportClassification(scenario string, matches []*factsv1.GenericFact) string {
	if len(matches) == 0 {
		return "missing"
	}
	importPath := normalizedImportPath(matches[0])
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
		got := normalizedImportPath(fact)
		if got == importPath {
			return true
		}
	}
	return false
}

func normalizedImportPath(fact *factsv1.GenericFact) string {
	return unquote(firstNonEmpty(
		fact.GetAttributes()["import_path"],
		fact.GetAttributes()["source_module"],
		fact.GetSubject(),
	))
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

func cleanEvidencePath(path string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
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
