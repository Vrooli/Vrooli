package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"proto-health/internal/protosurface"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type fakeLoader struct {
	surface   protosurface.Surface
	surfaces  map[string]protosurface.Surface
	err       error
	errors    map[string]error
	scenarios []string
}

func (f fakeLoader) LoadScenario(scenario string) (protosurface.Surface, error) {
	if f.errors != nil && f.errors[scenario] != nil {
		return protosurface.Surface{}, f.errors[scenario]
	}
	if f.surfaces != nil {
		if surface, ok := f.surfaces[scenario]; ok {
			return surface, nil
		}
	}
	return f.surface, f.err
}

func (f fakeLoader) ListScenarios() ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]string{}, f.scenarios...), nil
}

type fakeGenSyncChecker struct {
	status GenSyncStatus
	err    error
}

func (f fakeGenSyncChecker) CheckScenario(context.Context, string) (GenSyncStatus, error) {
	return f.status, f.err
}

type fakeCodeFactsClient struct {
	adoptionReport *factsv1.ProofReport
	adoptionErr    error
	endpointReport *factsv1.ProofReport
	endpointErr    error
	endpointIDs    []string
}

func (f *fakeCodeFactsClient) CheckProtoAdoption(context.Context, string) (*factsv1.ProofReport, error) {
	if f.adoptionErr != nil {
		return nil, f.adoptionErr
	}
	return f.adoptionReport, nil
}

func (f *fakeCodeFactsClient) CheckEndpointProof(_ context.Context, _ string, endpointIDs []string) (*factsv1.ProofReport, error) {
	f.endpointIDs = append([]string{}, endpointIDs...)
	if f.endpointErr != nil {
		return nil, f.endpointErr
	}
	return f.endpointReport, nil
}

func TestValidateScenarioCleanSurfacePasses(t *testing.T) {
	svc := New(Deps{Loader: fakeLoader{surface: cleanSurface()}})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Empty(t, report.Findings)
	require.Zero(t, report.Summary.Errors)
}

func TestValidateScenarioFindsGeneratedArtifactDrift(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		GenSyncChecker: fakeGenSyncChecker{status: GenSyncStatus{
			InSync: false,
			Drift:  []string{"packages/proto/gen/go/demo"},
			Detail: "1 generated slice differs after regeneration",
		}},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeGenOutOfSync, SeverityError)
}

func TestValidateScenarioFindsMissingCodeFactsProtoAdoption(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionReport: proofReport(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
				proofFact("proto_adoption:api", factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING, "api generated proto import missing"),
			),
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeProtoAdoptionMissing, SeverityError)
}

func TestValidateScenarioWarnsWhenCodeFactsUnavailable(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionErr: errors.New("code-facts not running"),
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeCodeFactsUnavailable, SeverityWarning)
}

func TestValidateScenarioIgnoresNonProofCodeFactsWarnings(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionReport: &factsv1.ProofReport{
				Family: factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
				Warnings: []*factsv1.Warning{{
					Code:    "typescript-code-graph.type_check_failure",
					Message: "src/App.tsx: [object Object]",
					Status:  factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
				}},
			},
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	for _, finding := range report.Findings {
		require.NotEqual(t, CodeProtoAdoptionUnsupported, finding.Code)
	}
}

func TestValidateScenarioFindsContradictedEndpointProof(t *testing.T) {
	client := &fakeCodeFactsClient{
		adoptionReport: proofReport(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
			proofFact("proto_adoption:api", factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "api imports generated proto"),
		),
		endpointReport: proofReport(factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
			proofFact("endpoint_proof:health", factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS, "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED, "health writes a different response proto"),
		),
	}
	svc := New(Deps{Loader: fakeLoader{surface: surfaceWithRESTException()}, CodeFacts: client})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	require.Equal(t, []string{"health"}, client.endpointIDs)
	requireFinding(t, report, CodeEndpointProofContradicted, SeverityError)
}

func TestValidateScenarioFindsMissingEndpointProof(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: surfaceWithRESTException()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionReport: proofReport(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION),
			endpointReport: proofReport(factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
				proofFact("endpoint_proof:health", factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS, "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING, "no payload writer found"),
			),
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeEndpointProofMissing, SeverityError)
}

func TestValidateScenarioWarnsForUnsupportedEndpointProof(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: surfaceWithRESTException()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionReport: proofReport(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION),
			endpointReport: proofReport(factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
				proofFact("endpoint_proof:health", factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS, "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED, "surface missing"),
			),
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeEndpointProofUnsupported, SeverityWarning)
}

func TestValidateScenarioAcceptsProvenCodeFactsEvidence(t *testing.T) {
	svc := New(Deps{
		Loader: fakeLoader{surface: surfaceWithRESTException()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionReport: proofReport(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
				proofFact("proto_adoption:api", factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "api imports generated proto"),
			),
			endpointReport: proofReport(factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
				proofFact("endpoint_proof:health", factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS, "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "health writes declared payload"),
			),
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	for _, finding := range report.Findings {
		require.NotContains(t, []string{
			CodeProtoAdoptionMissing,
			CodeProtoAdoptionContradicted,
			CodeEndpointProofMissing,
			CodeEndpointProofContradicted,
			CodeEndpointProofUnsupported,
		}, finding.Code)
	}
}

func TestValidateScenarioFindsPolicyViolations(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files,
		protosurface.File{
			Path:        "demo/v1/orders/orders.proto",
			Package:     "vrooli.demo.v1.orders",
			Version:     "v1",
			Domain:      "orders",
			Stability:   "experimental",
			Annotations: []protosurface.Annotation{{Name: "layer", Value: "5"}},
		},
		protosurface.File{
			Path:      "demo/v3/billing/billing.proto",
			Package:   "vrooli.demo.v3.wrong",
			Version:   "v3",
			Domain:    "billing",
			Stability: "stable",
		},
	)
	surface.Services = append(surface.Services,
		protosurface.Service{
			FilePath: "demo/v1/orders/orders.proto",
			Package:  "vrooli.demo.v1.orders",
			Name:     "OrdersService",
			FullName: "vrooli.demo.v1.orders.OrdersService",
			Domain:   "orders",
			RPCs: []protosurface.RPC{{
				Name:      "ListOrders",
				Input:     "vrooli.demo.v1.orders.ListOrdersRequest",
				Output:    "vrooli.demo.v1.orders.ListOrdersResponse",
				Transport: protosurface.TransportKindConnect,
			}},
		},
	)
	surface.Messages = append(surface.Messages,
		protosurface.Message{FilePath: "demo/v1/orders/orders.proto", Package: "vrooli.demo.v1.orders", Name: "ListOrdersRequest", FullName: "vrooli.demo.v1.orders.ListOrdersRequest", Domain: "orders"},
		protosurface.Message{FilePath: "demo/v1/orders/orders.proto", Package: "vrooli.demo.v1.orders", Name: "ListOrdersResponse", FullName: "vrooli.demo.v1.orders.ListOrdersResponse", Domain: "orders"},
		protosurface.Message{FilePath: "demo/v3/billing/billing.proto", Package: "vrooli.demo.v3.wrong", Name: "Unused", FullName: "vrooli.demo.v3.wrong.Unused", Domain: "billing"},
	)
	surface.IntraScenarioImports = append(surface.IntraScenarioImports,
		protosurface.Import{FromFile: "demo/v1/orders/orders.proto", ToFile: "demo/v3/billing/billing.proto", FromDomain: "orders", ToDomain: "billing"},
	)

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)

	requireFinding(t, report, CodePackageMismatch, SeverityError)
	requireFinding(t, report, CodeStabilityDishonest, SeverityError)
	requireFinding(t, report, CodeCrossDomainImport, SeverityWarning)
	requireFinding(t, report, CodeUnsupportedAnnotation, SeverityWarning)
	requireFinding(t, report, CodeVersionNaming, SeverityWarning)
}

func TestValidateScenarioFindsTemplateSource(t *testing.T) {
	surface := cleanSurface()
	surface.Files[0].Annotations = append(surface.Files[0].Annotations, protosurface.Annotation{Name: "template", Value: "react-vite/example"})

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeTemplateSource, SeverityWarning)
}

func TestValidateScenarioReportsStableMessageNotLocallyReachable(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/notes/attachments.proto",
		Package:   "vrooli.demo.v1.notes",
		Version:   "v1",
		Domain:    "notes",
		Stability: "stable",
	})
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/notes/attachments.proto",
		Package:  "vrooli.demo.v1.notes",
		Name:     "UploadAttachmentResponse",
		FullName: "vrooli.demo.v1.notes.UploadAttachmentResponse",
		Domain:   "notes",
	})

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodePossiblyUnused, SeverityInfo)
}

func TestValidateScenarioTreatsRESTExceptionResponseAsReachable(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/notes/attachments.proto",
		Package:   "vrooli.demo.v1.notes",
		Version:   "v1",
		Domain:    "notes",
		Stability: "stable",
	})
	surface.Messages = append(surface.Messages,
		protosurface.Message{
			FilePath: "demo/v1/notes/attachments.proto",
			Package:  "vrooli.demo.v1.notes",
			Name:     "Attachment",
			FullName: "vrooli.demo.v1.notes.Attachment",
			Domain:   "notes",
		},
		protosurface.Message{
			FilePath: "demo/v1/notes/attachments.proto",
			Package:  "vrooli.demo.v1.notes",
			Name:     "UploadAttachmentResponse",
			FullName: "vrooli.demo.v1.notes.UploadAttachmentResponse",
			Domain:   "notes",
			Fields: []protosurface.Field{{
				Name:        "attachment",
				Type:        "message",
				MessageType: "vrooli.demo.v1.notes.Attachment",
				Number:      1,
			}},
		},
	)
	surface.RESTExceptionRefs = []protosurface.RESTExceptionRef{{
		EndpointID: "notes_attach",
		Path:       "/api/v1/notes/{id}/attachments",
		Method:     "POST",
		Domain:     "notes",
		Message:    "UploadAttachmentResponse",
		FullName:   "vrooli.demo.v1.notes.UploadAttachmentResponse",
	}}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, CodePossiblyUnused, finding.Code, "REST-exception response messages should be reachable: %+v", report.Findings)
	}
}

func TestValidateScenarioRequiresRESTExceptionPayloadDeclarations(t *testing.T) {
	surface := cleanSurface()
	surface.RESTExceptions = []protosurface.RESTExceptionEndpoint{{
		EndpointID: "health",
		Path:       "/health",
		Method:     "GET",
		Domain:     "system",
		Reason:     "ops_probe",
	}}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeRESTPayloadMissingDeclaration, SeverityError)
}

func TestValidateScenarioFindsUnknownRESTExceptionPayloadMessage(t *testing.T) {
	surface := cleanSurface()
	surface.RESTExceptions = []protosurface.RESTExceptionEndpoint{{
		EndpointID:             "health",
		Path:                   "/health",
		Method:                 "GET",
		Domain:                 "system",
		Reason:                 "ops_probe",
		HasPayloadDeclarations: true,
	}}
	surface.RESTExceptionPayloads = []protosurface.RESTExceptionPayloadRef{
		{EndpointID: "health", Path: "/health", Method: "GET", Role: protosurface.RESTPayloadRoleRequest, Transport: "none", Conformance: "none", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
		{EndpointID: "health", Path: "/health", Method: "GET", Role: protosurface.RESTPayloadRoleResponse, ProtoFullName: "vrooli.demo.v1.health.Missing", Transport: "json", Conformance: "protojson", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
		{EndpointID: "health", Path: "/health", Method: "GET", Role: protosurface.RESTPayloadRoleError, Transport: "json", Conformance: "external_shape", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
	}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeRESTPayloadUnknownMessage, SeverityError)
}

func TestValidateScenarioFindsInvalidRESTExceptionConformance(t *testing.T) {
	surface := cleanSurface()
	surface.RESTExceptions = []protosurface.RESTExceptionEndpoint{{
		EndpointID:             "health",
		Path:                   "/health",
		Method:                 "GET",
		Domain:                 "system",
		Reason:                 "ops_probe",
		HasPayloadDeclarations: true,
	}}
	surface.RESTExceptionPayloads = []protosurface.RESTExceptionPayloadRef{
		{EndpointID: "health", Path: "/health", Method: "GET", Role: protosurface.RESTPayloadRoleRequest, Transport: "none", Conformance: "none", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
		{EndpointID: "health", Path: "/health", Method: "GET", Role: protosurface.RESTPayloadRoleResponse, Transport: "json", Conformance: "best_effort", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
		{EndpointID: "health", Path: "/health", Method: "GET", Role: protosurface.RESTPayloadRoleError, Transport: "json", Conformance: "external_shape", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
	}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeRESTPayloadInvalidConformance, SeverityError)
}

func TestValidateScenarioFindsUnknownImportKind(t *testing.T) {
	surface := cleanSurface()
	surface.IntraScenarioImports = []protosurface.Import{{
		FromFile:   "demo/v1/notes/notes.proto",
		ToFile:     "demo/v1/shared/errors.proto",
		FromDomain: "notes",
		ToDomain:   "shared",
	}}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireFinding(t, report, CodeImportKindUnknown, SeverityWarning)
}

func TestValidateScenarioFindsStableTransitiveDependencyMismatch(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files,
		protosurface.File{
			Path:      "demo/v1/notes/notes.proto",
			Package:   "vrooli.demo.v1.notes",
			Version:   "v1",
			Domain:    "notes",
			Stability: "stable",
		},
		protosurface.File{
			Path:      "demo/v1/shared/draft.proto",
			Package:   "vrooli.demo.v1.shared",
			Version:   "v1",
			Domain:    "shared",
			Stability: "experimental",
		},
	)
	surface.Services = []protosurface.Service{{
		FilePath: "demo/v1/notes/notes.proto",
		Package:  "vrooli.demo.v1.notes",
		Name:     "NotesService",
		FullName: "vrooli.demo.v1.notes.NotesService",
		Domain:   "notes",
		RPCs: []protosurface.RPC{{
			Name:      "GetNote",
			Input:     "vrooli.demo.v1.notes.GetNoteRequest",
			Output:    "vrooli.demo.v1.notes.GetNoteResponse",
			Transport: protosurface.TransportKindConnect,
		}},
	}}
	surface.Messages = []protosurface.Message{
		{FilePath: "demo/v1/notes/notes.proto", Package: "vrooli.demo.v1.notes", Name: "GetNoteRequest", FullName: "vrooli.demo.v1.notes.GetNoteRequest", Domain: "notes"},
		{
			FilePath: "demo/v1/notes/notes.proto",
			Package:  "vrooli.demo.v1.notes",
			Name:     "GetNoteResponse",
			FullName: "vrooli.demo.v1.notes.GetNoteResponse",
			Domain:   "notes",
			Fields: []protosurface.Field{{
				Name:        "draft",
				Type:        "message",
				MessageType: "vrooli.demo.v1.shared.DraftMetadata",
				Number:      1,
			}},
		},
		{FilePath: "demo/v1/shared/draft.proto", Package: "vrooli.demo.v1.shared", Name: "DraftMetadata", FullName: "vrooli.demo.v1.shared.DraftMetadata", Domain: "shared"},
	}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeStabilityDependencyMismatch, SeverityError)
}

func TestValidateScenarioFindsReusableTypeOutsideSharedDomain(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/errors/errors.proto",
		Package:   "vrooli.demo.v1.errors",
		Version:   "v1",
		Domain:    "errors",
		Stability: "stable",
	})
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/errors/errors.proto",
		Package:  "vrooli.demo.v1.errors",
		Name:     "ErrorEnvelope",
		FullName: "vrooli.demo.v1.errors.ErrorEnvelope",
		Domain:   "errors",
	})
	surface.RESTExceptionPayloads = []protosurface.RESTExceptionPayloadRef{
		{EndpointID: "health", Domain: "health", Role: protosurface.RESTPayloadRoleError, ProtoFullName: "vrooli.demo.v1.errors.ErrorEnvelope", Conformance: "protojson"},
		{EndpointID: "notes_attach", Domain: "notes", Role: protosurface.RESTPayloadRoleError, ProtoFullName: "vrooli.demo.v1.errors.ErrorEnvelope", Conformance: "protojson"},
	}

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeSharedTypeMisplaced, SeverityError)
}

func TestValidateScenarioFindsImportCycle(t *testing.T) {
	surface := cleanSurface()
	surface.IntraScenarioImports = []protosurface.Import{
		{FromFile: "demo/v1/shared/health.proto", ToFile: "demo/v1/shared/errors.proto", FromDomain: "shared", ToDomain: "shared"},
		{FromFile: "demo/v1/shared/errors.proto", ToFile: "demo/v1/shared/health.proto", FromDomain: "shared", ToDomain: "shared"},
	}
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/shared/errors.proto",
		Package:   "vrooli.demo.v1.shared",
		Version:   "v1",
		Domain:    "shared",
		Stability: "stable",
	})

	svc := New(Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireFinding(t, report, CodeCycle, SeverityError)
}

func TestDescribeScenarioProtosReturnsSurface(t *testing.T) {
	surface := cleanSurface()
	svc := New(Deps{Loader: fakeLoader{surface: surface}})

	got, err := svc.DescribeScenarioProtos(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, surface, got)
}

func TestDescribeScenariosProtosUsesExplicitSubset(t *testing.T) {
	demo := cleanSurface()
	demo.Scenario = "demo"
	other := cleanSurface()
	other.Scenario = "other"
	svc := New(Deps{Loader: fakeLoader{
		surfaces: map[string]protosurface.Surface{
			"demo":  demo,
			"other": other,
		},
	}})

	results, err := svc.DescribeScenariosProtos(context.Background(), []string{"demo", "other", "demo", ""}, 0, "")
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "demo", results[0].Scenario)
	require.Equal(t, demo, results[0].Surface)
	require.Equal(t, "other", results[1].Scenario)
	require.Equal(t, other, results[1].Surface)
}

func TestDescribeScenariosProtosListsAllAndAppliesLimit(t *testing.T) {
	svc := New(Deps{Loader: fakeLoader{
		scenarios: []string{"alpha", "beta", "gamma"},
		surfaces: map[string]protosurface.Surface{
			"alpha": {Scenario: "alpha"},
			"beta":  {Scenario: "beta"},
			"gamma": {Scenario: "gamma"},
		},
	}})

	results, err := svc.DescribeScenariosProtos(context.Background(), nil, 2, "")
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "alpha", results[0].Scenario)
	require.Equal(t, "beta", results[1].Scenario)
}

func TestDescribeScenariosProtosIsolatesPerScenarioErrors(t *testing.T) {
	svc := New(Deps{Loader: fakeLoader{
		surfaces: map[string]protosurface.Surface{
			"ok": {Scenario: "ok"},
		},
		errors: map[string]error{
			"bad": errors.New("no proto files found"),
		},
	}})

	results, err := svc.DescribeScenariosProtos(context.Background(), []string{"ok", "bad"}, 0, "")
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Empty(t, results[0].Error)
	require.Equal(t, "ok", results[0].Surface.Scenario)
	require.Equal(t, "bad", results[1].Scenario)
	require.Equal(t, "no proto files found", results[1].Error)
	require.Empty(t, results[1].Surface.Scenario)
}

func TestDescribeScenariosProtosFiltersByStability(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/experimental/preview.proto",
		Package:   "vrooli.demo.v1.experimental",
		Version:   "v1",
		Domain:    "experimental",
		Stability: "experimental",
	})
	surface.Messages = append(surface.Messages,
		protosurface.Message{FilePath: "demo/v1/experimental/preview.proto", Domain: "experimental", Name: "Preview"},
	)
	svc := New(Deps{Loader: fakeLoader{surface: surface}})

	results, err := svc.DescribeScenariosProtos(context.Background(), []string{"demo"}, 0, "experimental")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Surface.Files, 1)
	require.Equal(t, "demo/v1/experimental/preview.proto", results[0].Surface.Files[0].Path)
	require.Len(t, results[0].Surface.Messages, 1)
	require.Equal(t, "Preview", results[0].Surface.Messages[0].Name)
}

func TestDescribeScenariosProtosRejectsInvalidLimit(t *testing.T) {
	svc := New(Deps{Loader: fakeLoader{surface: cleanSurface()}})

	_, err := svc.DescribeScenariosProtos(context.Background(), []string{"demo"}, 501, "")
	require.ErrorContains(t, err, "limit must be between 0 and 500")
}

func cleanSurface() protosurface.Surface {
	return protosurface.Surface{
		Scenario:       "demo",
		TransportWorld: protosurface.TransportWorldConnect,
		Files: []protosurface.File{{
			Path:        "demo/v1/shared/health.proto",
			Package:     "vrooli.demo.v1.shared",
			Version:     "v1",
			Domain:      "shared",
			Stability:   "stable",
			Annotations: []protosurface.Annotation{{Name: "stability", Value: "stable"}},
		}},
	}
}

func surfaceWithRESTException() protosurface.Surface {
	surface := cleanSurface()
	surface.RESTExceptions = []protosurface.RESTExceptionEndpoint{{
		EndpointID:             "health",
		Path:                   "/health",
		Method:                 "GET",
		Domain:                 "health",
		Reason:                 "ops_probe",
		HasPayloadDeclarations: true,
	}}
	surface.RESTExceptionPayloads = []protosurface.RESTExceptionPayloadRef{
		{EndpointID: "health", Path: "/health", Method: "GET", Domain: "health", Role: protosurface.RESTPayloadRoleRequest, Transport: "none", Conformance: "none", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
		{EndpointID: "health", Path: "/health", Method: "GET", Domain: "health", Role: protosurface.RESTPayloadRoleResponse, ProtoFullName: "vrooli.demo.v1.shared.HealthResponse", Transport: "json", Conformance: "protojson", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
		{EndpointID: "health", Path: "/health", Method: "GET", Domain: "health", Role: protosurface.RESTPayloadRoleError, Transport: "json", Conformance: "external_shape", ProofStatus: protosurface.RESTPayloadProofNotEvaluated},
	}
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/shared/health.proto",
		Package:  "vrooli.demo.v1.shared",
		Name:     "HealthResponse",
		FullName: "vrooli.demo.v1.shared.HealthResponse",
		Domain:   "shared",
	})
	surface.RESTExceptionRefs = []protosurface.RESTExceptionRef{{
		EndpointID: "health",
		Path:       "/health",
		Method:     "GET",
		Domain:     "health",
		Message:    "HealthResponse",
		FullName:   "vrooli.demo.v1.shared.HealthResponse",
	}}
	return surface
}

func proofReport(family factsv1.FactFamily, facts ...*factsv1.GenericFact) *factsv1.ProofReport {
	return &factsv1.ProofReport{Family: family, Facts: facts}
}

func proofFact(id string, family factsv1.FactFamily, subject string, status factsv1.EvidenceStatus, message string) *factsv1.GenericFact {
	return &factsv1.GenericFact{
		Id:      id,
		Family:  family,
		Subject: subject,
		Evidence: []*factsv1.Evidence{{
			Status:  status,
			Message: message,
			Range:   &factsv1.SourceRange{File: "scenarios/demo/.vrooli/endpoints.json"},
		}},
	}
}

func requireFinding(t *testing.T, report Report, code string, severity Severity) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code && finding.Severity == severity {
			return
		}
	}
	require.Failf(t, "missing finding", "code=%s severity=%s findings=%+v", code, severity, report.Findings)
}
