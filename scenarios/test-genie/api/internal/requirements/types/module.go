package types

// RequirementModule represents a parsed requirement file (module.json or index.json).
type RequirementModule struct {
	Metadata     ModuleMetadata `json:"_metadata,omitempty"`
	Imports      []string       `json:"imports,omitempty"`
	Requirements []Requirement  `json:"requirements"`

	// Tracking fields (not serialized)
	FilePath     string `json:"-"`
	RelativePath string `json:"-"`
	IsIndex      bool   `json:"-"`
	ModuleName   string `json:"-"`
}

// ModuleMetadata contains module-level metadata.
type ModuleMetadata struct {
	Module          string `json:"module,omitempty"`
	ModuleName      string `json:"module_name,omitempty"`
	Description     string `json:"description,omitempty"`
	PRDRef          string `json:"prd_ref,omitempty"`
	Priority        string `json:"priority,omitempty"`
	LastValidatedAt string `json:"last_validated_at,omitempty"`
	AutoSyncEnabled *bool  `json:"auto_sync_enabled,omitempty"`
	SchemaVersion   string `json:"schema_version,omitempty"`
}

// GetModuleName returns the effective module name.
func (m *ModuleMetadata) GetModuleName() string {
	if m.ModuleName != "" {
		return m.ModuleName
	}
	return m.Module
}

// IsAutoSyncEnabled returns whether auto-sync is enabled (default true).
func (m *ModuleMetadata) IsAutoSyncEnabled() bool {
	if m.AutoSyncEnabled == nil {
		return true
	}
	return *m.AutoSyncEnabled
}

// Clone creates a deep copy of the module.
func (m *RequirementModule) Clone() *RequirementModule {
	if m == nil {
		return nil
	}

	clone := &RequirementModule{
		Metadata:     m.Metadata.Clone(),
		FilePath:     m.FilePath,
		RelativePath: m.RelativePath,
		IsIndex:      m.IsIndex,
		ModuleName:   m.ModuleName,
	}

	if len(m.Imports) > 0 {
		clone.Imports = make([]string, len(m.Imports))
		copy(clone.Imports, m.Imports)
	}

	if len(m.Requirements) > 0 {
		clone.Requirements = make([]Requirement, len(m.Requirements))
		for i, r := range m.Requirements {
			if cloned := r.Clone(); cloned != nil {
				clone.Requirements[i] = *cloned
			}
		}
	}

	return clone
}

// Clone creates a copy of the metadata.
func (m ModuleMetadata) Clone() ModuleMetadata {
	clone := ModuleMetadata{
		Module:          m.Module,
		ModuleName:      m.ModuleName,
		Description:     m.Description,
		PRDRef:          m.PRDRef,
		Priority:        m.Priority,
		LastValidatedAt: m.LastValidatedAt,
		SchemaVersion:   m.SchemaVersion,
	}

	if m.AutoSyncEnabled != nil {
		val := *m.AutoSyncEnabled
		clone.AutoSyncEnabled = &val
	}

	return clone
}

// RequirementCount returns the total number of requirements in the module.
func (m *RequirementModule) RequirementCount() int {
	if m == nil {
		return 0
	}
	return len(m.Requirements)
}

// GetRequirement finds a requirement by ID within the module.
func (m *RequirementModule) GetRequirement(id string) *Requirement {
	if m == nil {
		return nil
	}
	for i := range m.Requirements {
		if m.Requirements[i].ID == id {
			return &m.Requirements[i]
		}
	}
	return nil
}

// AllRequirementIDs returns all requirement IDs in the module.
func (m *RequirementModule) AllRequirementIDs() []string {
	if m == nil {
		return nil
	}
	ids := make([]string, len(m.Requirements))
	for i, r := range m.Requirements {
		ids[i] = r.ID
	}
	return ids
}

// HasRequirements returns true if the module contains at least one requirement.
func (m *RequirementModule) HasRequirements() bool {
	return m != nil && len(m.Requirements) > 0
}

// HasImports returns true if the module has any imports.
func (m *RequirementModule) HasImports() bool {
	return m != nil && len(m.Imports) > 0
}

// EffectiveName returns the display name for the module.
func (m *RequirementModule) EffectiveName() string {
	if m == nil {
		return ""
	}
	if m.ModuleName != "" {
		return m.ModuleName
	}
	if name := m.Metadata.GetModuleName(); name != "" {
		return name
	}
	return m.RelativePath
}
