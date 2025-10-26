package settings

// Settings validation limits
const (
	// Slot constraints
	MinSlots = 1
	MaxSlots = 5

	// Refresh interval constraints (seconds)
	MinRefreshInterval = 5
	MaxRefreshInterval = 300

	// Max turns constraints
	MinMaxTurns = 5
	MaxMaxTurns = 80

	// Task timeout constraints (minutes)
	MinTaskTimeout = 5
	MaxTaskTimeout = 240

	// Recycler interval constraints (seconds)
	MinRecyclerInterval = 30
	MaxRecyclerInterval = 1800

	// Recycler max generations constraints
	MinRecyclerMaxGenerations = 1
	MaxRecyclerMaxGenerations = 10

	// Recycler batch size constraints
	MinRecyclerBatchSize = 1
	MaxRecyclerBatchSize = 10

	// Recycler limit constraints
	MinRecyclerLimit = 1
	MaxRecyclerLimit = 100
)

// Default settings values
const (
	DefaultSlots           = 1
	DefaultRefreshInterval = 30
	DefaultMaxTurns        = 40
	DefaultTaskTimeout     = 60
	DefaultAllowedTools    = "all"
	DefaultSkipPermissions = true
	DefaultActive          = false

	// Recycler defaults
	DefaultRecyclerEnabledFor     = "off"
	DefaultRecyclerInterval       = 300
	DefaultRecyclerMaxGenerations = 3
	DefaultRecyclerBatchSize      = 5
	DefaultRecyclerLimit          = 50
	DefaultRecyclerModelProvider  = "anthropic"
	DefaultRecyclerModelID        = "claude-3-5-sonnet-20241022"
)
