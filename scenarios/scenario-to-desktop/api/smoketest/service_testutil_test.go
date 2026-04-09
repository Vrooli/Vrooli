package smoketest_test

import (
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

type testServiceDeps struct {
	store              *mocks.MockStore
	cancelManager      *mocks.MockCancelManager
	telemetryIngestor  *mocks.MockTelemetryIngestor
	config             smoketest.Config
	executor           *mocks.MockProcessExecutor
	platformResolver   *mocks.MockPlatformResolver
	telemetryResolver  *mocks.MockTelemetryPathResolver
	outputParser       *mocks.MockOutputParser
	fs                 *mocks.MockFileSystem
	logger             *mocks.MockLogger
	port               int
	telemetryExtractor *mocks.MockTelemetryErrorExtractor
}

func createTestService(configure func(*testServiceDeps)) *smoketest.DefaultService {
	deps := &testServiceDeps{
		store:              mocks.NewMockStore(),
		cancelManager:      mocks.NewMockCancelManager(),
		telemetryIngestor:  mocks.NewMockTelemetryIngestor(),
		config:             smoketest.DefaultConfig(),
		executor:           mocks.NewMockProcessExecutor(),
		platformResolver:   mocks.NewMockPlatformResolver(),
		telemetryResolver:  mocks.NewMockTelemetryPathResolver(),
		outputParser:       mocks.NewMockOutputParser(),
		fs:                 mocks.NewMockFileSystem(),
		logger:             mocks.NewMockLogger(),
		port:               8080,
		telemetryExtractor: mocks.NewMockTelemetryErrorExtractor(),
	}

	if configure != nil {
		configure(deps)
	}

	return smoketest.NewService(
		deps.store,
		deps.cancelManager,
		deps.telemetryIngestor,
		deps.config,
		deps.executor,
		deps.platformResolver,
		deps.telemetryResolver,
		deps.outputParser,
		deps.fs,
		deps.logger,
		deps.port,
		deps.telemetryExtractor,
	)
}

// ValidStateTransitions defines all valid state transitions in the smoke test state machine.
var ValidStateTransitions = map[smoketest.State][]smoketest.State{
	"": { // Initial empty state
		smoketest.StateInitializing,
	},
	smoketest.StateInitializing: {
		smoketest.StateValidatingArtifact,
		smoketest.StateFailed,
	},
	smoketest.StateValidatingArtifact: {
		smoketest.StateResolvingCommand,
		smoketest.StateFailed,
	},
	smoketest.StateResolvingCommand: {
		smoketest.StateExecuting,
		smoketest.StateFailed,
	},
	smoketest.StateExecuting: {
		smoketest.StateParsingOutput,
		smoketest.StateFailed,
	},
	smoketest.StateParsingOutput: {
		smoketest.StateTelemetryUpload,
		smoketest.StateTelemetryFallback,
		smoketest.StatePassed,
		smoketest.StateFailed,
	},
	smoketest.StateTelemetryUpload: {
		smoketest.StatePassed,
		smoketest.StateFailed,
	},
	smoketest.StateTelemetryFallback: {
		smoketest.StatePassed,
		smoketest.StateFailed,
	},
	smoketest.StatePassed: {}, // Terminal state
	smoketest.StateFailed: {}, // Terminal state
}

// isValidTransition checks if a state transition is valid according to the state machine.
func isValidTransition(from, to smoketest.State) bool {
	validTargets, ok := ValidStateTransitions[from]
	if !ok {
		return false
	}
	for _, valid := range validTargets {
		if valid == to {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// logsContain checks if any log entry contains the given substring.
func logsContain(logs []string, substr string) bool {
	for _, log := range logs {
		if contains(log, substr) {
			return true
		}
	}
	return false
}
