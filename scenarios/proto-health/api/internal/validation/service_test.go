package validation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"

	"proto-health/internal/protosurface"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func newTestService(t *testing.T, deps Deps) *Service {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	catalog, err := NewFindingCatalog(spec)
	require.NoError(t, err)
	deps.Catalog = catalog
	return New(deps)
}

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
	adoptionCtx    context.Context
	endpointReport *factsv1.ProofReport
	endpointErr    error
	endpointIDs    []string
}

func (f *fakeCodeFactsClient) CheckProtoAdoption(ctx context.Context, _ string) (*factsv1.ProofReport, error) {
	f.adoptionCtx = ctx
	if f.adoptionErr != nil {
		return nil, f.adoptionErr
	}
	return f.adoptionReport, nil
}

func TestValidateScenarioBoundsOptionalCodeFactsEvidence(t *testing.T) {
	client := &fakeCodeFactsClient{adoptionErr: context.DeadlineExceeded}
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: cleanSurface()}, CodeFacts: client})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireFinding(t, report, CodeCodeFactsUnavailable, SeverityWarning)
	require.NotNil(t, client.adoptionCtx)
	deadline, ok := client.adoptionCtx.Deadline()
	require.True(t, ok, "optional proof RPC must receive a bounded context")
	require.LessOrEqual(t, time.Until(deadline), codeFactsBudget)
}

func (f *fakeCodeFactsClient) CheckEndpointProof(_ context.Context, _ string, endpointIDs []string) (*factsv1.ProofReport, error) {
	f.endpointIDs = append([]string{}, endpointIDs...)
	if f.endpointErr != nil {
		return nil, f.endpointErr
	}
	return f.endpointReport, nil
}

func TestValidateScenarioCleanSurfacePasses(t *testing.T) {
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: cleanSurface()}})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Empty(t, report.Findings)
	require.Zero(t, report.Summary.Errors)
}

func TestValidateScenarioFindsGeneratedArtifactDrift(t *testing.T) {
	svc := newTestService(t, Deps{
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

func TestValidateScenarioFindsMissingGenerationManifest(t *testing.T) {
	svc := newTestService(t, Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		GenSyncChecker: fakeGenSyncChecker{status: GenSyncStatus{
			ManifestMissing: true,
			Drift:           []string{"packages/proto/gen/manifests/demo.lock.json"},
		}},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeGenManifestMissing, SeverityError)
}

func TestValidateScenarioWarnsOnGenerationToolchainDrift(t *testing.T) {
	svc := newTestService(t, Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		GenSyncChecker: fakeGenSyncChecker{status: GenSyncStatus{
			InSync:         true,
			ToolchainDrift: true,
		}},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeGenToolchainDrift, SeverityWarning)
}

func TestValidateScenarioFindsMissingCodeFactsProtoAdoption(t *testing.T) {
	svc := newTestService(t, Deps{
		Loader: fakeLoader{surface: cleanSurface()},
		CodeFacts: &fakeCodeFactsClient{
			adoptionReport: proofReport(factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
				proofFact("proto_adoption:api", factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING, "api generated proto import missing"),
			),
		},
	})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeProtoAdoptionMissing, SeverityWarning)
}

func TestValidateScenarioWarnsWhenCodeFactsUnavailable(t *testing.T) {
	svc := newTestService(t, Deps{
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
	svc := newTestService(t, Deps{
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
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surfaceWithRESTException()}, CodeFacts: client})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	require.Equal(t, []string{"health"}, client.endpointIDs)
	requireFinding(t, report, CodeEndpointProofContradicted, SeverityError)
}

func TestValidateScenarioFindsMissingEndpointProof(t *testing.T) {
	svc := newTestService(t, Deps{
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
	require.True(t, report.Passed)
	requireFinding(t, report, CodeEndpointProofMissing, SeverityWarning)
}

func TestValidateScenarioWarnsForUnsupportedEndpointProof(t *testing.T) {
	svc := newTestService(t, Deps{
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
	svc := newTestService(t, Deps{
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)

	requireFinding(t, report, CodePackageMismatch, SeverityError)
	requireFinding(t, report, CodeStabilityDishonest, SeverityError)
	requireFinding(t, report, CodeCrossDomainImport, SeverityWarning)
	requireFinding(t, report, CodeUnsupportedAnnotation, SeverityWarning)
	requireFinding(t, report, CodeVersionNaming, SeverityWarning)
}

func TestValidateScenarioAcceptsConstraintAnnotation(t *testing.T) {
	surface := cleanSurface()
	surface.Files[0].Annotations = append(surface.Files[0].Annotations, protosurface.Annotation{Name: "constraint", Value: "one of: healthy, degraded"})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireNoFinding(t, report, CodeUnsupportedAnnotation)
	requireNoFinding(t, report, CodeConstraintMissingProtovalidate)
}

func TestValidateScenarioWarnsWhenFieldConstraintLacksProtovalidate(t *testing.T) {
	surface := cleanSurface()
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/shared/health.proto",
		Package:  "vrooli.demo.v1.shared",
		Name:     "HealthRequest",
		FullName: "vrooli.demo.v1.shared.HealthRequest",
		Domain:   "shared",
		Fields: []protosurface.Field{{
			Name:        "status",
			Type:        "string",
			Number:      1,
			Annotations: []protosurface.Annotation{{Name: "constraint", Value: "one of: healthy, degraded"}},
		}},
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeConstraintMissingProtovalidate, SeverityWarning)
}

func TestValidateScenarioAcceptsFieldConstraintWithProtovalidate(t *testing.T) {
	surface := cleanSurface()
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/shared/health.proto",
		Package:  "vrooli.demo.v1.shared",
		Name:     "HealthRequest",
		FullName: "vrooli.demo.v1.shared.HealthRequest",
		Domain:   "shared",
		Fields: []protosurface.Field{{
			Name:               "status",
			Type:               "string",
			Number:             1,
			Annotations:        []protosurface.Annotation{{Name: "constraint", Value: "one of: healthy, degraded"}},
			HasValidationRules: true,
		}},
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireNoFinding(t, report, CodeConstraintMissingProtovalidate)
}

func TestValidateScenarioWarnsWhenMessageConstraintLacksProtovalidate(t *testing.T) {
	surface := cleanSurface()
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath:    "demo/v1/shared/health.proto",
		Package:     "vrooli.demo.v1.shared",
		Name:        "HealthRequest",
		FullName:    "vrooli.demo.v1.shared.HealthRequest",
		Domain:      "shared",
		Annotations: []protosurface.Annotation{{Name: "constraint", Value: "status and reason must agree"}},
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeConstraintMissingProtovalidate, SeverityWarning)
}

func TestValidateScenarioFindsTemplateSource(t *testing.T) {
	surface := cleanSurface()
	surface.Files[0].Annotations = append(surface.Files[0].Annotations, protosurface.Annotation{Name: "template", Value: "react-vite/example"})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodeTemplateSource, SeverityWarning)
}

func TestValidateScenarioKeepsTemplateSourceForUndivergedTemplate(t *testing.T) {
	root := t.TempDir()
	templatePath := filepath.Join(root, "templates", "scenarios", "react-vite", "proto", "v1", "shared", "health.proto")
	scenarioPath := filepath.Join(root, "packages", "proto", "schemas", "demo", "v1", "shared", "health.proto")
	template := []byte("syntax = \"proto3\";\npackage vrooli.{{SCENARIO_ID_SNAKE}}.v1.shared;\nmessage Response {}\n")
	require.NoError(t, os.MkdirAll(filepath.Dir(templatePath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(scenarioPath), 0o755))
	require.NoError(t, os.WriteFile(templatePath, template, 0o644))
	require.NoError(t, os.WriteFile(scenarioPath, []byte("syntax = \"proto3\";\npackage vrooli.demo.v1.shared;\nmessage Response {}\n"), 0o644))

	surface := cleanSurface()
	surface.Files[0].Annotations = append(surface.Files[0].Annotations, protosurface.Annotation{Name: "template", Value: "react-vite/example"})
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}, RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireFinding(t, report, CodeTemplateSource, SeverityWarning)
}

func TestValidateScenarioClearsTemplateSourceAfterContentDiverges(t *testing.T) {
	root := t.TempDir()
	templatePath := filepath.Join(root, "templates", "scenarios", "react-vite", "proto", "v1", "shared", "health.proto")
	scenarioPath := filepath.Join(root, "packages", "proto", "schemas", "demo", "v1", "shared", "health.proto")
	require.NoError(t, os.MkdirAll(filepath.Dir(templatePath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(scenarioPath), 0o755))
	require.NoError(t, os.WriteFile(templatePath, []byte("syntax = \"proto3\";\npackage vrooli.{{SCENARIO_ID_SNAKE}}.v1.shared;\nmessage Response {}\n"), 0o644))
	require.NoError(t, os.WriteFile(scenarioPath, []byte("syntax = \"proto3\";\npackage vrooli.demo.v1.shared;\nmessage ScenarioHealthResponse {}\n"), 0o644))

	surface := cleanSurface()
	surface.Files[0].Annotations = append(surface.Files[0].Annotations, protosurface.Annotation{Name: "template", Value: "react-vite/example"})
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}, RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireNoFinding(t, report, CodeTemplateSource)
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodePossiblyUnused, SeverityWarning)
}

func TestValidateScenarioExemptsExperimentalUnservedMessageFromPossiblyUnused(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/preview/preview.proto",
		Package:   "vrooli.demo.v1.preview",
		Version:   "v1",
		Domain:    "preview",
		Stability: "experimental",
	})
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/preview/preview.proto",
		Package:  "vrooli.demo.v1.preview",
		Name:     "PreviewEnvelope",
		FullName: "vrooli.demo.v1.preview.PreviewEnvelope",
		Domain:   "preview",
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireNoFinding(t, report, CodePossiblyUnused)
}

func TestValidateScenarioUsesFleetReachabilityForPossiblyUnused(t *testing.T) {
	producer := surfaceWithExportedMessage("producer")
	consumer := cleanSurface()
	consumer.Scenario = "consumer"
	consumer.Files = append(consumer.Files, protosurface.File{
		Path:      "consumer/v1/notes/notes.proto",
		Package:   "vrooli.consumer.v1.notes",
		Version:   "v1",
		Domain:    "notes",
		Stability: "stable",
	})
	consumer.Services = []protosurface.Service{{
		FilePath: "consumer/v1/notes/notes.proto",
		Package:  "vrooli.consumer.v1.notes",
		Name:     "NotesService",
		FullName: "vrooli.consumer.v1.notes.NotesService",
		Domain:   "notes",
		RPCs: []protosurface.RPC{{
			Name:      "ListNotes",
			Input:     "vrooli.consumer.v1.notes.ListNotesRequest",
			Output:    "vrooli.consumer.v1.notes.ListNotesResponse",
			Transport: protosurface.TransportKindConnect,
		}},
	}}
	consumer.Messages = append(consumer.Messages,
		protosurface.Message{FilePath: "consumer/v1/notes/notes.proto", Package: "vrooli.consumer.v1.notes", Name: "ListNotesRequest", FullName: "vrooli.consumer.v1.notes.ListNotesRequest", Domain: "notes"},
		protosurface.Message{
			FilePath: "consumer/v1/notes/notes.proto",
			Package:  "vrooli.consumer.v1.notes",
			Name:     "ListNotesResponse",
			FullName: "vrooli.consumer.v1.notes.ListNotesResponse",
			Domain:   "notes",
			Fields: []protosurface.Field{{
				Name:        "producer_item",
				Type:        "message",
				MessageType: "vrooli.producer.v1.shared.ProducerItem",
				Number:      1,
			}},
		},
	)
	consumer.CrossScenarioImports = []protosurface.Import{{
		FromScenario: "consumer",
		ToScenario:   "producer",
		FromFile:     "consumer/v1/notes/notes.proto",
		ToFile:       "producer/v1/shared/items.proto",
		Kind:         protosurface.ImportKindCrossScenario,
	}}

	svc := newTestService(t, Deps{Loader: fakeLoader{
		scenarios: []string{"producer", "consumer"},
		surfaces: map[string]protosurface.Surface{
			"producer": producer,
			"consumer": consumer,
		},
	}})
	report, err := svc.ValidateScenario(context.Background(), "producer")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireNoFinding(t, report, CodePossiblyUnused)
}

func TestValidateScenarioReportsPossiblyUnusedWithoutFleetConsumer(t *testing.T) {
	producer := surfaceWithExportedMessage("producer")
	svc := newTestService(t, Deps{Loader: fakeLoader{
		scenarios: []string{"producer"},
		surfaces:  map[string]protosurface.Surface{"producer": producer},
	}})

	report, err := svc.ValidateScenario(context.Background(), "producer")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireFinding(t, report, CodePossiblyUnused, SeverityWarning)
}

func TestValidateScenarioAcceptsRetainedExternalStableMessage(t *testing.T) {
	surface := surfaceWithExportedMessage("producer")
	surface.Messages[0].Annotations = []protosurface.Annotation{{Name: "see", Value: "external:published-api"}}
	svc := newTestService(t, Deps{Loader: fakeLoader{
		scenarios: []string{"producer"},
		surfaces:  map[string]protosurface.Surface{"producer": surface},
	}})

	report, err := svc.ValidateScenario(context.Background(), "producer")
	require.NoError(t, err)
	requireNoFinding(t, report, CodePossiblyUnused)
}

func TestValidateScenarioReflagsRetainedConsumerWhenDrifted(t *testing.T) {
	producer := surfaceWithExportedMessage("producer")
	producer.Messages[0].Annotations = []protosurface.Annotation{{Name: "see", Value: "consumer:consumer"}}
	consumer := cleanSurface()
	consumer.Scenario = "consumer"

	svc := newTestService(t, Deps{Loader: fakeLoader{
		scenarios: []string{"producer", "consumer"},
		surfaces: map[string]protosurface.Surface{
			"producer": producer,
			"consumer": consumer,
		},
	}})
	report, err := svc.ValidateScenario(context.Background(), "producer")
	require.NoError(t, err)
	requireFinding(t, report, CodePossiblyUnused, SeverityWarning)
}

func TestValidateScenarioExemptsConventionalSharedEnvelopeFromPossiblyUnused(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files, protosurface.File{
		Path:      "demo/v1/shared/errors.proto",
		Package:   "vrooli.demo.v1.shared",
		Version:   "v1",
		Domain:    "shared",
		Stability: "stable",
	})
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: "demo/v1/shared/errors.proto",
		Package:  "vrooli.demo.v1.shared",
		Name:     "ErrorEnvelope",
		FullName: "vrooli.demo.v1.shared.ErrorEnvelope",
		Domain:   "shared",
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireNoFinding(t, report, CodePossiblyUnused)
}

func TestValidateScenarioExemptsConventionalHealthAndErrorMessagesFromPossiblyUnused(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files,
		protosurface.File{Path: "demo/v1/health/health.proto", Package: "vrooli.demo.v1.health", Version: "v1", Domain: "health", Stability: "stable"},
		protosurface.File{Path: "demo/v1/errors/errors.proto", Package: "vrooli.demo.v1.errors", Version: "v1", Domain: "errors", Stability: "stable"},
	)
	surface.Messages = append(surface.Messages,
		protosurface.Message{FilePath: "demo/v1/health/health.proto", Package: "vrooli.demo.v1.health", Name: "Response", FullName: "vrooli.demo.v1.health.Response", Domain: "health"},
		protosurface.Message{FilePath: "demo/v1/health/health.proto", Package: "vrooli.demo.v1.health", Name: "DependencyStatus", FullName: "vrooli.demo.v1.health.DependencyStatus", Domain: "health"},
		protosurface.Message{FilePath: "demo/v1/errors/errors.proto", Package: "vrooli.demo.v1.errors", Name: "ErrorEnvelope", FullName: "vrooli.demo.v1.errors.ErrorEnvelope", Domain: "errors"},
	)

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.True(t, report.Passed)
	requireNoFinding(t, report, CodePossiblyUnused)
}

func TestCheckDomainMismatchExemptsConventionalHealthAndErrorsDomains(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "demo", "api", "handlers", "health"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "demo", "api", "handlers", "errors"), 0o755))
	svc := &Service{repoRoot: root}
	surface := protosurface.Surface{
		Scenario: "demo",
		Files: []protosurface.File{
			{Path: "demo/v1/health/health.proto", Domain: "health"},
			{Path: "demo/v1/errors/errors.proto", Domain: "errors"},
		},
	}

	require.Empty(t, svc.checkDomainMismatch(surface))
}

func TestValidateScenarioSkipsMapEntryPossiblyUnused(t *testing.T) {
	surface := cleanSurface()
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath:   "demo/v1/notes/notes.proto",
		Package:    "vrooli.demo.v1.notes",
		Name:       "LabelsEntry",
		FullName:   "vrooli.demo.v1.notes.Note.LabelsEntry",
		Domain:     "notes",
		IsMapEntry: true,
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, CodePossiblyUnused, finding.Code)
	}
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, CodePossiblyUnused, finding.Code, "REST-exception response messages should be reachable: %+v", report.Findings)
	}
}

func TestValidateScenarioRequiresRESTExceptionPayloadDeclarations(t *testing.T) {
	surface := cleanSurface()
	surface.RESTExceptions = []protosurface.RESTExceptionEndpoint{{
		EndpointID: "notes_export",
		Path:       "/api/v1/notes/export",
		Method:     "GET",
		Domain:     "notes",
		Reason:     "third_party_shape",
	}}

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeRESTPayloadMissingDeclaration, SeverityError)
}

func TestValidateScenarioExemptsConventionalInfraRESTExceptionPayloadDeclarations(t *testing.T) {
	surface := cleanSurface()
	surface.RESTExceptions = []protosurface.RESTExceptionEndpoint{
		{EndpointID: "health", Path: "/health", Method: "GET", Domain: "system", Reason: "ops_probe"},
		{EndpointID: "jwks", Path: "/.well-known/jwks.json", Method: "GET", Domain: "auth", Reason: "third_party_shape"},
	}

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, CodeRESTPayloadMissingDeclaration, finding.Code, "infra endpoints should not require payload declarations: %+v", report.Findings)
	}
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeSharedTypeMisplaced, SeverityError)
}

func TestValidateScenarioSuppressesCrossDomainImportCoveredBySharedTypePlacement(t *testing.T) {
	surface := cleanSurface()
	surface.Files = append(surface.Files,
		protosurface.File{Path: "demo/v1/runs/runs.proto", Package: "vrooli.demo.v1.runs", Version: "v1", Domain: "runs", Stability: "stable"},
		protosurface.File{Path: "demo/v1/presence/presence.proto", Package: "vrooli.demo.v1.presence", Version: "v1", Domain: "presence", Stability: "stable"},
	)
	surface.Messages = append(surface.Messages,
		protosurface.Message{FilePath: "demo/v1/runs/runs.proto", Package: "vrooli.demo.v1.runs", Name: "RunEvent", FullName: "vrooli.demo.v1.runs.RunEvent", Domain: "runs"},
		protosurface.Message{
			FilePath: "demo/v1/presence/presence.proto",
			Package:  "vrooli.demo.v1.presence",
			Name:     "Heartbeat",
			FullName: "vrooli.demo.v1.presence.Heartbeat",
			Domain:   "presence",
			Fields: []protosurface.Field{{
				Name:        "event",
				Type:        "message",
				MessageType: "vrooli.demo.v1.runs.RunEvent",
				Number:      1,
			}},
		},
	)
	surface.Services = append(surface.Services, protosurface.Service{
		FilePath: "demo/v1/runs/runs.proto",
		Package:  "vrooli.demo.v1.runs",
		Name:     "RunsService",
		FullName: "vrooli.demo.v1.runs.RunsService",
		Domain:   "runs",
		RPCs: []protosurface.RPC{{
			Name:      "WatchRuns",
			Input:     "vrooli.demo.v1.runs.RunEvent",
			Output:    "vrooli.demo.v1.runs.RunEvent",
			Transport: protosurface.TransportKindConnect,
		}},
	})
	surface.IntraScenarioImports = append(surface.IntraScenarioImports, protosurface.Import{
		FromFile:   "demo/v1/presence/presence.proto",
		ToFile:     "demo/v1/runs/runs.proto",
		FromDomain: "presence",
		ToDomain:   "runs",
		Kind:       protosurface.ImportKindScenarioLocal,
	})

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeSharedTypeMisplaced, SeverityError)
	requireNoFinding(t, report, CodeCrossDomainImport)
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

	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})
	report, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
	requireFinding(t, report, CodeCycle, SeverityError)
}

func TestDescribeScenarioProtosReturnsSurface(t *testing.T) {
	surface := cleanSurface()
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})

	got, err := svc.DescribeScenarioProtos(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, surface, got)
}

func TestDescribeScenariosProtosUsesExplicitSubset(t *testing.T) {
	demo := cleanSurface()
	demo.Scenario = "demo"
	other := cleanSurface()
	other.Scenario = "other"
	svc := newTestService(t, Deps{Loader: fakeLoader{
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
	svc := newTestService(t, Deps{Loader: fakeLoader{
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
	svc := newTestService(t, Deps{Loader: fakeLoader{
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
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: surface}})

	results, err := svc.DescribeScenariosProtos(context.Background(), []string{"demo"}, 0, "experimental")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Surface.Files, 1)
	require.Equal(t, "demo/v1/experimental/preview.proto", results[0].Surface.Files[0].Path)
	require.Len(t, results[0].Surface.Messages, 1)
	require.Equal(t, "Preview", results[0].Surface.Messages[0].Name)
}

func TestDescribeScenariosProtosRejectsInvalidLimit(t *testing.T) {
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: cleanSurface()}})

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

func surfaceWithExportedMessage(scenario string) protosurface.Surface {
	surface := cleanSurface()
	surface.Scenario = scenario
	surface.Files[0].Path = scenario + "/v1/shared/health.proto"
	surface.Files[0].Package = "vrooli." + scenario + ".v1.shared"
	surface.Files = append(surface.Files, protosurface.File{
		Path:      scenario + "/v1/shared/items.proto",
		Package:   "vrooli." + scenario + ".v1.shared",
		Version:   "v1",
		Domain:    "shared",
		Stability: "stable",
	})
	surface.Messages = append(surface.Messages, protosurface.Message{
		FilePath: scenario + "/v1/shared/items.proto",
		Package:  "vrooli." + scenario + ".v1.shared",
		Name:     "ProducerItem",
		FullName: "vrooli." + scenario + ".v1.shared.ProducerItem",
		Domain:   "shared",
	})
	return surface
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

func requireNoFinding(t *testing.T, report Report, code string) {
	t.Helper()
	for _, finding := range report.Findings {
		require.NotEqual(t, code, finding.Code, "unexpected finding: %+v", report.Findings)
	}
}
