package executionwriter

import (
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/config"
)

// SetArtifactConfig updates the writer-wide fallback collection settings.
// Per-execution configuration is preferred for concurrent runs.
func (r *FileWriter) SetArtifactConfig(cfg *config.ArtifactCollectionSettings) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cfg == nil {
		r.artifactConfig = config.DefaultArtifactSettings()
		return
	}
	r.artifactConfig = *cfg
}

func (r *FileWriter) GetArtifactConfig() config.ArtifactCollectionSettings {
	if r == nil {
		return config.DefaultArtifactSettings()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.artifactConfig
}

// SetArtifactConfigForExecution prevents one concurrent execution's artifact
// profile from leaking into another execution using the same writer.
func (r *FileWriter) SetArtifactConfigForExecution(executionID uuid.UUID, cfg *config.ArtifactCollectionSettings) {
	if r == nil {
		return
	}
	if cfg == nil {
		r.perExecConfig.Delete(executionID.String())
		return
	}
	r.perExecConfig.Store(executionID.String(), *cfg)
}

func (r *FileWriter) ForgetExecution(executionID uuid.UUID) {
	if r != nil {
		r.perExecConfig.Delete(executionID.String())
	}
}

func (r *FileWriter) artifactConfigForExecution(executionID uuid.UUID) config.ArtifactCollectionSettings {
	if r == nil {
		return config.DefaultArtifactSettings()
	}
	if value, ok := r.perExecConfig.Load(executionID.String()); ok {
		if cfg, ok := value.(config.ArtifactCollectionSettings); ok {
			return cfg
		}
	}
	return r.GetArtifactConfig()
}
