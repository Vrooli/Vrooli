package settings

// Settings validation limits
const (
	// Slot constraints
	MinSlots = 1
	MaxSlots = 5

	// Cooldown constraints (seconds)
	MinCooldownSeconds = 5
	MaxCooldownSeconds = 300

	// Max turns constraints
	MinMaxTurns = 5
	MaxMaxTurns = 100

	// Task timeout constraints (minutes)
	MinTaskTimeout = 5
	MaxTaskTimeout = 240

	// Idle timeout cap constraints (minutes)
	MinIdleTimeoutCap = 2
	MaxIdleTimeoutCap = 240

	// Recycler interval constraints (seconds)
	MinRecyclerInterval = 30
	MaxRecyclerInterval = 1800

	// Recycler retry constraints
	MinRecyclerMaxRetries     = 0
	MaxRecyclerMaxRetries     = 10
	MinRecyclerRetryDelaySecs = 1
	MaxRecyclerRetryDelaySecs = 300

	// Recycler max generations constraints
	MinRecyclerMaxGenerations = 1
	MaxRecyclerMaxGenerations = 10

	// Recycler batch size constraints
	MinRecyclerBatchSize = 1
	MaxRecyclerBatchSize = 10

	// Recycler limit constraints
	MinRecyclerLimit = 1
	MaxRecyclerLimit = 100

	// Recycler threshold constraints
	MinRecyclerCompletionThreshold = 1
	MaxRecyclerCompletionThreshold = 10
	MinRecyclerFailureThreshold    = 1
	MaxRecyclerFailureThreshold    = 10
)

// Default settings values
const (
	DefaultSlots           = 1
	DefaultCooldownSeconds = 30
	DefaultMaxTurns        = 80
	DefaultTaskTimeout     = 30
	DefaultIdleTimeoutCap  = DefaultTaskTimeout
	DefaultAllowedTools    = "Read,Write,Edit,Bash,LS,Glob,Grep"
	DefaultSkipPermissions = true
	DefaultActive          = false
	DefaultCondensedMode   = false

	// Recycler defaults
	DefaultRecyclerEnabledFor          = "off"
	DefaultRecyclerInterval            = 300
	DefaultRecyclerMaxRetries          = 3
	DefaultRecyclerRetryDelaySeconds   = 2
	DefaultRecyclerMaxGenerations      = 3
	DefaultRecyclerBatchSize           = 5
	DefaultRecyclerLimit               = 50
	DefaultRecyclerModelProvider       = "ollama"
	DefaultRecyclerModelID             = "llama3.1:8b"
	DefaultRecyclerCompletionThreshold = 3
	DefaultRecyclerFailureThreshold    = 5
)
