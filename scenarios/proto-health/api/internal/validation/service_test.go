package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"proto-health/internal/protosurface"
)

type fakeLoader struct {
	surface protosurface.Surface
	err     error
}

func (f fakeLoader) LoadScenario(string) (protosurface.Surface, error) {
	return f.surface, f.err
}

type fakeGenSyncChecker struct {
	status GenSyncStatus
	err    error
}

func (f fakeGenSyncChecker) CheckScenario(context.Context, string) (GenSyncStatus, error) {
	return f.status, f.err
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

func requireFinding(t *testing.T, report Report, code string, severity Severity) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code && finding.Severity == severity {
			return
		}
	}
	require.Failf(t, "missing finding", "code=%s severity=%s findings=%+v", code, severity, report.Findings)
}
