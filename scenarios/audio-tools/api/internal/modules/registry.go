// Package modules is the single registration point for the scenario's
// API modules' static metadata. Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate
// domains uniformly.
//
// The runtime Module(...) constructors stay inline in main.go's
// server.New(...) call — they need live deps (db handle, clock, logger)
// and abstracting them is needless ceremony. This package only handles
// the static side: the Endpoints slice each handler exports for
// codegen, and the Schema() function each handler re-exports for
// EnsureSchemas.
package modules

import (
	"audio-tools/internal/modulekit"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	audioH "audio-tools/handlers/audio"
	diagH "audio-tools/handlers/diagnostics"
	healthH "audio-tools/handlers/health"
	hsH "audio-tools/handlers/health_status"
	measuresH "audio-tools/handlers/measures"
	modelsH "audio-tools/handlers/models"
	plH "audio-tools/handlers/provider_lifecycle"
	sessionH "audio-tools/handlers/session"
	settingsH "audio-tools/handlers/settings"
	sttH "audio-tools/handlers/stt"
	summarizeH "audio-tools/handlers/summarize"
	ttsH "audio-tools/handlers/tts"
	usageH "audio-tools/handlers/usage"
	"audio-tools/internal/byokstore"
	localdb "audio-tools/internal/database"
	"audio-tools/internal/duplex"
	trustfloor "audio-tools/internal/qualification"
	intsettings "audio-tools/internal/settings"
	intstt "audio-tools/internal/stt"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/measures"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
	sessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
func AllEndpoints() []modulekit.EndpointDescriptor {
	out := make([]modulekit.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, hsH.Endpoints...)
	out = append(out, plH.Endpoints...)
	out = append(out, modelsH.Endpoints...)
	out = append(out, measuresH.Endpoints...)
	out = append(out, audioH.Endpoints...)
	out = append(out, diagH.Endpoints...)
	out = append(out, sessionH.Endpoints...)
	out = append(out, settingsH.Endpoints...)
	out = append(out, sttH.Endpoints...)
	out = append(out, summarizeH.Endpoints...)
	out = append(out, ttsH.Endpoints...)
	out = append(out, usageH.Endpoints...)
	return appendLabEndpoints(out)
}

// LabEndpoints lists routes that exist only for corpus curation, experiments,
// or qualification runs. Product runtimes can mount ProductEndpoints without
// exposing this operator surface.
func LabEndpoints() []modulekit.EndpointDescriptor {
	var out []modulekit.EndpointDescriptor
	for _, endpoint := range AllEndpoints() {
		switch endpoint.Category {
		case "corpus", "experiment", "validation":
			out = append(out, endpoint)
		}
	}
	return out
}

// ProductEndpoints is the serving and operator configuration API surface.
func ProductEndpoints() []modulekit.EndpointDescriptor {
	lab := make(map[string]struct{})
	for _, endpoint := range LabEndpoints() {
		lab[endpoint.ID] = struct{}{}
	}
	var out []modulekit.EndpointDescriptor
	for _, endpoint := range AllEndpoints() {
		if _, isLab := lab[endpoint.ID]; !isLab {
			out = append(out, endpoint)
		}
	}
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the scenario-owned domain proto descriptors, in
// registration order. The externally owned scenario-validation provider
// protocol is declared in AllEndpoints but is not treated as an Audio Tools
// domain proto surface here.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "audio", File: audiov1.File_audio_tools_v1_audio_audio_proto},
		{Module: "diagnostics", File: diagv1.File_audio_tools_v1_diagnostics_diagnostics_proto},
		{Module: "health_status", File: hsv1.File_audio_tools_v1_health_status_health_status_proto},
		{Module: "models", File: modelsv1.File_audio_tools_v1_models_models_proto},
		{Module: "measures", File: measuresv1.File_audio_tools_v1_measures_measures_proto},
		{Module: "provider_lifecycle", File: plv1.File_audio_tools_v1_provider_lifecycle_provider_lifecycle_proto},
		{Module: "session", File: sessv1.File_audio_tools_v1_session_session_proto},
		{Module: "settings", File: settv1.File_audio_tools_v1_settings_settings_proto},
		{Module: "stt", File: sttv1.File_audio_tools_v1_stt_stt_proto},
		{Module: "summarize", File: summv1.File_audio_tools_v1_summarize_summarize_proto},
		{Module: "tts", File: ttsv1.File_audio_tools_v1_tts_tts_proto},
		{Module: "usage", File: usagev1.File_audio_tools_v1_usage_usage_proto},
	}
}

// ProductSchemas returns schemas needed by the serving and operator
// configuration API. Lab storage is kept separate so product bootstrap can
// later avoid creating corpus and experiment tables.
func ProductSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(byokstore.Schema),
		apidb.SchemaProviderFunc(duplex.Schema),
		apidb.SchemaProviderFunc(intsettings.Schema),
		apidb.SchemaProviderFunc(usagereport.Schema),
		apidb.SchemaProviderFunc(intstt.Schema),
		apidb.SchemaProviderFunc(inttts.Schema),
		apidb.SchemaProviderFunc(trustfloor.Schema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(hsH.Schema),
		apidb.SchemaProviderFunc(plH.Schema),
		apidb.SchemaProviderFunc(modelsH.Schema),
		apidb.SchemaProviderFunc(audioH.Schema),
		apidb.SchemaProviderFunc(diagH.Schema),
		apidb.SchemaProviderFunc(sessionH.Schema),
		apidb.SchemaProviderFunc(settingsH.Schema),
		apidb.SchemaProviderFunc(sttH.Schema),
		apidb.SchemaProviderFunc(summarizeH.Schema),
		apidb.SchemaProviderFunc(ttsH.Schema),
		apidb.SchemaProviderFunc(usageH.Schema),
	}
}

// LabSchemas returns storage owned by corpus curation and experiments.
func LabSchemas() []apidb.SchemaProvider {
	return labSchemas()
}

// AllSchemas is the complete migration inventory used by shared deployments
// and tools. Product-only bootstraps should use ProductSchemas instead.
func AllSchemas() []apidb.SchemaProvider {
	return appendLabSchemas(ProductSchemas())
}
