// Package settings owns the per-principal UI/CLI preferences row. The
// table is single-row in v1 (principal_id always 'local'); multi-tenant
// support is out of scope per the flow-verifier PRD. Defaults are
// served when the row does not yet exist so a fresh database returns a
// usable settings payload on first GET.
package settings

import "time"

// PrincipalLocal is the only principal id v1 ever writes. The column is
// kept on the table so multi-tenant support is a column-population task
// rather than a migration when it eventually lands.
const PrincipalLocal = "local"

// Theme enumerates the three theme modes the UI exposes.
type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeSystem Theme = "system"
)

// FontScale enumerates the three font-scale steps.
type FontScale string

const (
	FontScaleSm FontScale = "sm"
	FontScaleMd FontScale = "md"
	FontScaleLg FontScale = "lg"
)

// Density enumerates the two list/table density modes.
type Density string

const (
	DensityComfortable Density = "comfortable"
	DensityCompact     Density = "compact"
)

// InventoryFilters is the typed shape of the persisted inventory filter
// state. Stored as JSON TEXT in user_settings.inventory_filters and
// validated by the service before write.
type InventoryFilters struct {
	Search   string             `json:"search"`
	Language string             `json:"language"`
	Status   []string           `json:"status"`
	Sort     InventorySortOrder `json:"sort"`
}

// InventorySortOrder is the column/direction pair used by the inventory
// page's sort header. Direction is always "asc" or "desc".
type InventorySortOrder struct {
	Key string `json:"key"`
	Dir string `json:"dir"`
}

// Settings is the in-memory shape exposed to handlers and the CLI. JSON
// tags are camelCase to match the wire contract in plan §9. The
// inventory_filters column is decoded into the typed struct here so
// callers never juggle raw JSON strings.
type Settings struct {
	PrincipalID      string           `json:"principalId"`
	Theme            Theme            `json:"theme"`
	FontScale        FontScale        `json:"fontScale"`
	ReducedMotion    bool             `json:"reducedMotion"`
	RTL              bool             `json:"rtl"`
	DefaultRoot      string           `json:"defaultRoot"`
	Density          Density          `json:"density"`
	SidebarWidth     int              `json:"sidebarWidth"`
	InventoryFilters InventoryFilters `json:"inventoryFilters"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

// Patch is a partial update body. Pointer fields distinguish "omitted"
// from "set to zero value" — only non-nil fields are merged into the
// existing row by Service.Upsert.
type Patch struct {
	Theme            *Theme            `json:"theme,omitempty"`
	FontScale        *FontScale        `json:"fontScale,omitempty"`
	ReducedMotion    *bool             `json:"reducedMotion,omitempty"`
	RTL              *bool             `json:"rtl,omitempty"`
	DefaultRoot      *string           `json:"defaultRoot,omitempty"`
	Density          *Density          `json:"density,omitempty"`
	SidebarWidth     *int              `json:"sidebarWidth,omitempty"`
	InventoryFilters *InventoryFilters `json:"inventoryFilters,omitempty"`
}

// ValidationError is returned by Service.Upsert when a Patch carries an
// unknown enum value or out-of-range integer. Handlers translate this
// to HTTP 400 + invalid_request.
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Message != "" {
		return "settings: " + e.Field + "=" + e.Value + ": " + e.Message
	}
	return "settings: invalid " + e.Field + "=" + e.Value
}

// DefaultSettings returns the in-memory defaults served when the row
// does not yet exist. Mirrors the table-level DEFAULT clauses in
// schema.sql; the two MUST stay in lockstep so a freshly-inserted row
// round-trips byte-identically to a hard-coded GET response.
func DefaultSettings() Settings {
	return Settings{
		PrincipalID:   PrincipalLocal,
		Theme:         ThemeSystem,
		FontScale:     FontScaleMd,
		ReducedMotion: false,
		RTL:           false,
		DefaultRoot:   ".",
		Density:       DensityComfortable,
		SidebarWidth:  320,
		InventoryFilters: InventoryFilters{
			Search:   "",
			Language: "all",
			Status:   []string{},
			Sort:     InventorySortOrder{Key: "flowId", Dir: "asc"},
		},
	}
}
