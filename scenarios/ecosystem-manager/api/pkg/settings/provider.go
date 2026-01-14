package settings

// SettingsProvider defines an interface for accessing application settings.
// This abstraction enables unit testing without relying on global state.
type SettingsProvider interface {
	// GetSettings returns a copy of the current settings.
	GetSettings() Settings

	// IsActive returns whether the processor should be active.
	IsActive() bool

	// GetRecyclerSettings returns current recycler configuration.
	GetRecyclerSettings() RecyclerSettings

	// GetSlots returns the number of concurrent execution slots.
	GetSlots() int

	// GetCooldownSeconds returns the cooldown period between task executions.
	GetCooldownSeconds() int

	// GetTaskTimeout returns the task execution timeout in minutes.
	GetTaskTimeout() int

	// GetMaxTurns returns the maximum agent turns per execution.
	GetMaxTurns() int
}

// DefaultSettingsProvider wraps the global settings functions to implement SettingsProvider.
// This is used in production where global state is acceptable.
type DefaultSettingsProvider struct{}

// Compile-time assertion that DefaultSettingsProvider implements SettingsProvider.
var _ SettingsProvider = (*DefaultSettingsProvider)(nil)

// NewDefaultSettingsProvider creates a new DefaultSettingsProvider.
func NewDefaultSettingsProvider() *DefaultSettingsProvider {
	return &DefaultSettingsProvider{}
}

// GetSettings returns a copy of the current settings from global state.
func (p *DefaultSettingsProvider) GetSettings() Settings {
	return GetSettings()
}

// IsActive returns whether the processor should be active from global state.
func (p *DefaultSettingsProvider) IsActive() bool {
	return IsActive()
}

// GetRecyclerSettings returns current recycler configuration from global state.
func (p *DefaultSettingsProvider) GetRecyclerSettings() RecyclerSettings {
	return GetRecyclerSettings()
}

// GetSlots returns the number of concurrent execution slots.
func (p *DefaultSettingsProvider) GetSlots() int {
	return GetSettings().Slots
}

// GetCooldownSeconds returns the cooldown period between task executions.
func (p *DefaultSettingsProvider) GetCooldownSeconds() int {
	return GetSettings().CooldownSeconds
}

// GetTaskTimeout returns the task execution timeout in minutes.
func (p *DefaultSettingsProvider) GetTaskTimeout() int {
	return GetSettings().TaskTimeout
}

// GetMaxTurns returns the maximum agent turns per execution.
func (p *DefaultSettingsProvider) GetMaxTurns() int {
	return GetSettings().MaxTurns
}

// MockSettingsProvider is a test double for SettingsProvider.
// It allows configuring return values for testing.
type MockSettingsProvider struct {
	MockSettings         Settings
	MockIsActive         bool
	MockRecyclerSettings RecyclerSettings
}

// Compile-time assertion that MockSettingsProvider implements SettingsProvider.
var _ SettingsProvider = (*MockSettingsProvider)(nil)

// NewMockSettingsProvider creates a MockSettingsProvider with sensible defaults.
func NewMockSettingsProvider() *MockSettingsProvider {
	return &MockSettingsProvider{
		MockSettings:         newDefaultSettings(),
		MockIsActive:         true,
		MockRecyclerSettings: newDefaultSettings().Recycler,
	}
}

// GetSettings returns the mock settings.
func (p *MockSettingsProvider) GetSettings() Settings {
	return p.MockSettings
}

// IsActive returns the mock active state.
func (p *MockSettingsProvider) IsActive() bool {
	return p.MockIsActive
}

// GetRecyclerSettings returns the mock recycler settings.
func (p *MockSettingsProvider) GetRecyclerSettings() RecyclerSettings {
	return p.MockRecyclerSettings
}

// GetSlots returns the number of concurrent execution slots from mock settings.
func (p *MockSettingsProvider) GetSlots() int {
	return p.MockSettings.Slots
}

// GetCooldownSeconds returns the cooldown period from mock settings.
func (p *MockSettingsProvider) GetCooldownSeconds() int {
	return p.MockSettings.CooldownSeconds
}

// GetTaskTimeout returns the task execution timeout from mock settings.
func (p *MockSettingsProvider) GetTaskTimeout() int {
	return p.MockSettings.TaskTimeout
}

// GetMaxTurns returns the maximum agent turns from mock settings.
func (p *MockSettingsProvider) GetMaxTurns() int {
	return p.MockSettings.MaxTurns
}
