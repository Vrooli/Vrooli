package dbdetect

import (
	"test-genie/internal/orchestrator/workspace"
)

// seam: Manifest — boundary between dbdetect and the on-disk service.json
// schema, so collectors can be tested with a fake instead of writing JSON.
type Manifest interface {
	Resources() []ManifestResource
	SQLitePathEnvVars() []string
}

// ManifestResource is the projection of one service-manifest resource entry
// that dbdetect cares about.
type ManifestResource struct {
	Key      string
	Type     string
	Required bool
	Enabled  bool
}

// WrapManifest adapts a loaded *workspace.ServiceManifest to the dbdetect
// Manifest seam. A nil input yields an empty manifest (no resources,
// no env vars) so callers do not need to special-case missing files.
func WrapManifest(m *workspace.ServiceManifest) Manifest {
	return manifestWrap{m: m}
}

type manifestWrap struct {
	m *workspace.ServiceManifest
}

func (w manifestWrap) Resources() []ManifestResource {
	if w.m == nil || w.m.Dependencies.Resources == nil {
		return nil
	}
	out := make([]ManifestResource, 0, len(w.m.Dependencies.Resources))
	for key, r := range w.m.Dependencies.Resources {
		out = append(out, ManifestResource{
			Key:      key,
			Type:     r.Type,
			Required: r.Required,
			Enabled:  r.Enabled,
		})
	}
	return out
}

func (w manifestWrap) SQLitePathEnvVars() []string {
	if w.m == nil {
		return nil
	}
	return w.m.SQLitePathEnvVars()
}
