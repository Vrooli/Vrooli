package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"test-genie-api/database"
	"test-genie-api/models"
)

// TestOrchestrator handles multi-phase test execution
type TestOrchestrator struct {
	mu               sync.Mutex
	activeExecutions map[uuid.UUID]*ExecutionContext
	workerPool       chan struct{}
}

// ExecutionContext maintains the state of a test execution
type ExecutionContext struct {
	ExecutionID  uuid.UUID
	SuiteID      uuid.UUID
	TestCases    []models.TestCase
	Status       string
	StartTime    time.Time
	EndTime      *time.Time
	Results      []models.TestResult
	Errors       []error
	CancelFunc   context.CancelFunc
	NotifyConfig models.NotificationSettings
}

// VaultExecutionContext maintains the state of a vault execution
type VaultExecutionContext struct {
	ExecutionID     uuid.UUID
	VaultID         uuid.UUID
	Vault           *models.TestVault
	CurrentPhase    string
	CompletedPhases []string
	FailedPhases    []string
	PhaseResults    map[string]*models.PhaseResult
	Status          string
	StartTime       time.Time
	CancelFunc      context.CancelFunc
}

var (
	defaultOrchestrator *TestOrchestrator
	once                sync.Once
)

// GetOrchestrator returns the singleton orchestrator instance
func GetOrchestrator() *TestOrchestrator {
	once.Do(func() {
		defaultOrchestrator = &TestOrchestrator{
			activeExecutions: make(map[uuid.UUID]*ExecutionContext),
			workerPool:       make(chan struct{}, 10), // Max 10 concurrent test executions
		}
		// Initialize worker pool
		for i := 0; i < 10; i++ {
			defaultOrchestrator.workerPool <- struct{}{}
		}
	})
	return defaultOrchestrator
}

// ExecuteTestSuite orchestrates the execution of a test suite
func (o *TestOrchestrator) ExecuteTestSuite(ctx context.Context, suiteID uuid.UUID, config models.ExecuteTestSuiteRequest) (*models.ExecuteTestSuiteResponse, error) {
	// Get the test suite from database
	suite, err := database.GetTestSuite(suiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test suite: %w", err)
	}

	// Create execution context
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)

	executionID := uuid.New()
	execution := &ExecutionContext{
		ExecutionID:  executionID,
		SuiteID:      suiteID,
		TestCases:    suite.TestCases,
		Status:       "queued",
		StartTime:    time.Now(),
		Results:      []models.TestResult{},
		Errors:       []error{},
		CancelFunc:   cancel,
		NotifyConfig: config.NotificationSettings,
	}

	// Store execution context
	o.mu.Lock()
	o.activeExecutions[executionID] = execution
	o.mu.Unlock()

	// Store initial execution in database
	dbExecution := &models.TestExecution{
		ID:            executionID,
		SuiteID:       suiteID,
		ExecutionType: config.ExecutionType,
		StartTime:     execution.StartTime,
		Status:        "queued",
		Environment:   config.Environment,
		Results:       []models.TestResult{},
		PerformanceMetrics: models.PerformanceMetrics{
			ExecutionTime: 0,
			ResourceUsage: make(map[string]interface{}),
			ErrorCount:    0,
		},
	}

	if err := database.StoreTestExecution(dbExecution); err != nil {
		log.Printf("Warning: Failed to store initial execution: %v", err)
	}

	// Start execution asynchronously
	go o.runTestExecution(execCtx, execution, config.ParallelExecution)

	// Return response immediately
	response := &models.ExecuteTestSuiteResponse{
		ExecutionID:       executionID,
		Status:            "started",
		EstimatedDuration: float64(len(suite.TestCases) * 5), // Estimate 5 seconds per test
		TestCount:         len(suite.TestCases),
		TrackingURL:       fmt.Sprintf("/api/v1/test-execution/%s/results", executionID),
	}

	return response, nil
}

// runTestExecution runs the actual test execution
func (o *TestOrchestrator) runTestExecution(ctx context.Context, execution *ExecutionContext, parallel bool) {
	// Acquire worker slot
	<-o.workerPool
	defer func() {
		o.workerPool <- struct{}{}
	}()

	// Update status to running
	execution.Status = "running"

	log.Printf("🚀 Starting test execution %s with %d test cases", execution.ExecutionID, len(execution.TestCases))

	// Execute tests
	if parallel {
		o.runTestsParallel(ctx, execution)
	} else {
		o.runTestsSequential(ctx, execution)
	}

	// Calculate final status
	endTime := time.Now()
	execution.EndTime = &endTime

	failedCount := 0
	passedCount := 0
	for _, result := range execution.Results {
		if result.Status == "failed" {
			failedCount++
		} else if result.Status == "passed" {
			passedCount++
		}
	}

	if failedCount > 0 {
		execution.Status = "failed"
	} else if passedCount == len(execution.TestCases) {
		execution.Status = "completed"
	} else {
		execution.Status = "partial"
	}

	// Update database
	database.UpdateExecutionStatus(execution.ExecutionID, execution.Status, "")

	// Send notifications if configured
	if execution.NotifyConfig.OnCompletion || (execution.NotifyConfig.OnFailure && execution.Status == "failed") {
		o.sendNotification(execution)
	}

	log.Printf("✅ Test execution %s completed: %s (passed: %d, failed: %d)",
		execution.ExecutionID, execution.Status, passedCount, failedCount)
}

// runTestsSequential runs tests one by one
func (o *TestOrchestrator) runTestsSequential(ctx context.Context, execution *ExecutionContext) {
	for _, testCase := range execution.TestCases {
		select {
		case <-ctx.Done():
			log.Printf("Test execution %s cancelled", execution.ExecutionID)
			execution.Status = "cancelled"
			return
		default:
			result := o.executeTestCase(ctx, testCase, execution.ExecutionID)
			execution.Results = append(execution.Results, result)

			// Store result in database
			if err := database.StoreTestResult(&result); err != nil {
				log.Printf("Warning: Failed to store test result: %v", err)
			}
		}
	}
}

// runTestsParallel runs tests concurrently
func (o *TestOrchestrator) runTestsParallel(ctx context.Context, execution *ExecutionContext) {
	var wg sync.WaitGroup
	resultsChan := make(chan models.TestResult, len(execution.TestCases))
	semaphore := make(chan struct{}, 5) // Max 5 concurrent tests

	for _, testCase := range execution.TestCases {
		wg.Add(1)
		go func(tc models.TestCase) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()

				result := o.executeTestCase(ctx, tc, execution.ExecutionID)
				resultsChan <- result

				// Store result in database
				if err := database.StoreTestResult(&result); err != nil {
					log.Printf("Warning: Failed to store test result: %v", err)
				}
			}
		}(testCase)
	}

	// Wait for all tests to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		execution.Results = append(execution.Results, result)
	}
}

// executeTestCase executes a single test case
func (o *TestOrchestrator) executeTestCase(ctx context.Context, testCase models.TestCase, executionID uuid.UUID) models.TestResult {
	result := models.TestResult{
		ID:                  uuid.New(),
		ExecutionID:         executionID,
		TestCaseID:          testCase.ID,
		TestCaseName:        testCase.Name,
		TestCaseDescription: testCase.Description,
		StartedAt:           time.Now(),
		Assertions:          []models.AssertionResult{},
		Artifacts:           make(map[string]interface{}),
	}

	// Create timeout context for individual test
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(testCase.Timeout)*time.Second)
	defer cancel()

	// Execute the test based on type
	switch testCase.TestType {
	case "unit":
		o.executeUnitTest(testCtx, &testCase, &result)
	case "integration":
		o.executeIntegrationTest(testCtx, &testCase, &result)
	case "performance":
		o.executePerformanceTest(testCtx, &testCase, &result)
	default:
		o.executeGenericTest(testCtx, &testCase, &result)
	}

	// Calculate duration
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt).Seconds()

	return result
}

// executeUnitTest executes a unit test
func (o *TestOrchestrator) executeUnitTest(ctx context.Context, testCase *models.TestCase, result *models.TestResult) {
	// Simulate test execution - in real implementation, this would run actual test code
	log.Printf("Executing unit test: %s", testCase.Name)

	// For now, simulate with a shell command
	cmd := exec.CommandContext(ctx, "bash", "-c", testCase.TestCode)
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Status = "failed"
		errorMsg := fmt.Sprintf("Test execution failed: %v\nOutput: %s", err, string(output))
		result.ErrorMessage = &errorMsg
	} else {
		result.Status = "passed"
	}

	// Add assertion results
	result.Assertions = append(result.Assertions, models.AssertionResult{
		Name:     "test_execution",
		Expected: "success",
		Actual:   result.Status,
		Passed:   result.Status == "passed",
		Message:  string(output),
	})
}

// executeIntegrationTest executes an integration test
func (o *TestOrchestrator) executeIntegrationTest(ctx context.Context, testCase *models.TestCase, result *models.TestResult) {
	log.Printf("Executing integration test: %s", testCase.Name)

	// Integration tests might need to set up test environments
	// For now, similar to unit test but with more complex setup
	cmd := exec.CommandContext(ctx, "bash", "-c", testCase.TestCode)
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Status = "failed"
		errorMsg := fmt.Sprintf("Integration test failed: %v\nOutput: %s", err, string(output))
		result.ErrorMessage = &errorMsg
	} else {
		result.Status = "passed"
	}
}

// executePerformanceTest executes a performance test
func (o *TestOrchestrator) executePerformanceTest(ctx context.Context, testCase *models.TestCase, result *models.TestResult) {
	log.Printf("Executing performance test: %s", testCase.Name)

	startTime := time.Now()

	// Execute performance test
	cmd := exec.CommandContext(ctx, "bash", "-c", testCase.TestCode)
	output, err := cmd.CombinedOutput()

	executionTime := time.Since(startTime)

	// Store performance metrics in artifacts
	result.Artifacts["execution_time_ms"] = executionTime.Milliseconds()
	result.Artifacts["output"] = string(output)

	if err != nil {
		result.Status = "failed"
		errorMsg := fmt.Sprintf("Performance test failed: %v", err)
		result.ErrorMessage = &errorMsg
	} else {
		// Check if performance meets criteria (example: must complete in < 1 second)
		if executionTime < 1*time.Second {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			errorMsg := fmt.Sprintf("Performance test exceeded time limit: %v", executionTime)
			result.ErrorMessage = &errorMsg
		}
	}
}

// executeGenericTest executes a generic test
func (o *TestOrchestrator) executeGenericTest(ctx context.Context, testCase *models.TestCase, result *models.TestResult) {
	log.Printf("Executing generic test: %s", testCase.Name)

	// For generic tests, just run the test code
	cmd := exec.CommandContext(ctx, "bash", "-c", testCase.TestCode)
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Status = "failed"
		errorMsg := fmt.Sprintf("Test failed: %v\nOutput: %s", err, string(output))
		result.ErrorMessage = &errorMsg
	} else {
		result.Status = "passed"
	}
}

// ExecuteTestVault orchestrates multi-phase vault execution
func (o *TestOrchestrator) ExecuteTestVault(ctx context.Context, vaultID uuid.UUID, config models.ExecuteTestSuiteRequest) (*models.VaultExecution, error) {
	// Get the vault from database
	vault, err := database.GetTestVault(vaultID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test vault: %w", err)
	}

	// Create vault execution
	execution := &models.VaultExecution{
		ID:              uuid.New(),
		VaultID:         vaultID,
		ExecutionType:   config.ExecutionType,
		StartTime:       time.Now(),
		Status:          "running",
		CurrentPhase:    vault.Phases[0],
		CompletedPhases: []string{},
		FailedPhases:    []string{},
		PhaseResults:    make(map[string]models.PhaseResult),
		Environment:     config.Environment,
	}

	// Execute phases sequentially
	for _, phase := range vault.Phases {
		log.Printf("🔄 Executing vault phase: %s", phase)
		execution.CurrentPhase = phase

		phaseResult := o.executeVaultPhase(ctx, vault, phase)
		execution.PhaseResults[phase] = phaseResult

		if phaseResult.Status == "failed" {
			execution.FailedPhases = append(execution.FailedPhases, phase)

			// Check if failure is critical
			if vault.SuccessCriteria.NoCriticalFailures {
				execution.Status = "failed"
				break
			}
		} else {
			execution.CompletedPhases = append(execution.CompletedPhases, phase)
		}
	}

	// Determine final status
	if len(execution.CompletedPhases) == len(vault.Phases) {
		execution.Status = "completed"
	} else if len(execution.FailedPhases) > 0 {
		execution.Status = "failed"
	} else {
		execution.Status = "partial"
	}

	endTime := time.Now()
	execution.EndTime = &endTime

	return execution, nil
}

// executeVaultPhase executes a single phase of a vault
func (o *TestOrchestrator) executeVaultPhase(ctx context.Context, vault *models.TestVault, phaseName string) models.PhaseResult {
	phaseConfig, exists := vault.PhaseConfigurations[phaseName]
	if !exists {
		return models.PhaseResult{
			PhaseName:    phaseName,
			Status:       "failed",
			StartTime:    time.Now(),
			ErrorMessage: "Phase configuration not found",
		}
	}

	result := models.PhaseResult{
		PhaseName:   phaseName,
		Status:      "running",
		StartTime:   time.Now(),
		TestResults: []models.TestResult{},
		Metrics:     make(map[string]interface{}),
	}

	// Create phase timeout
	phaseCtx, cancel := context.WithTimeout(ctx, time.Duration(phaseConfig.Timeout)*time.Second)
	defer cancel()

	// Execute phase tests
	for _, test := range phaseConfig.Tests {
		testResult := o.executePhaseTest(phaseCtx, test)
		result.TestResults = append(result.TestResults, testResult)

		if testResult.Status == "failed" {
			result.Status = "failed"
		}
	}

	// If no failures, mark as completed
	if result.Status != "failed" {
		result.Status = "completed"
	}

	endTime := time.Now()
	result.EndTime = &endTime

	return result
}

// executePhaseTest executes a test within a vault phase
func (o *TestOrchestrator) executePhaseTest(ctx context.Context, test models.PhaseTest) models.TestResult {
	result := models.TestResult{
		ID:                  uuid.New(),
		TestCaseName:        test.Name,
		TestCaseDescription: test.Description,
		StartedAt:           time.Now(),
		Assertions:          []models.AssertionResult{},
		Artifacts:           make(map[string]interface{}),
	}

	// Execute test steps
	for _, step := range test.Steps {
		if err := o.executePhaseStep(ctx, step); err != nil {
			result.Status = "failed"
			errorMsg := err.Error()
			result.ErrorMessage = &errorMsg
			break
		}
	}

	if result.Status != "failed" {
		result.Status = "passed"
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt).Seconds()

	return result
}

// executePhaseStep executes a single step in a phase test
func (o *TestOrchestrator) executePhaseStep(ctx context.Context, step models.PhaseStep) error {
	// Execute based on action type
	switch step.Action {
	case "shell":
		cmd := exec.CommandContext(ctx, "bash", "-c", step.Expected)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("shell command failed: %v\nOutput: %s", err, string(output))
		}
	case "http":
		// Would implement HTTP request logic here
		log.Printf("Executing HTTP action: %v", step.Config)
	case "validate":
		// Would implement validation logic here
		log.Printf("Executing validation: %s", step.Expected)
	default:
		log.Printf("Unknown action type: %s", step.Action)
	}

	return nil
}

// sendNotification sends test completion notifications
func (o *TestOrchestrator) sendNotification(execution *ExecutionContext) {
	if execution.NotifyConfig.WebhookURL == "" {
		return
	}

	// Would implement webhook notification here
	log.Printf("📧 Sending notification for execution %s to %s",
		execution.ExecutionID, execution.NotifyConfig.WebhookURL)
}

// GetExecutionStatus returns the current status of an execution
func (o *TestOrchestrator) GetExecutionStatus(executionID uuid.UUID) (*ExecutionContext, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	execution, exists := o.activeExecutions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	return execution, nil
}

// CancelExecution cancels a running execution
func (o *TestOrchestrator) CancelExecution(executionID uuid.UUID) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	execution, exists := o.activeExecutions[executionID]
	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	if execution.CancelFunc != nil {
		execution.CancelFunc()
		execution.Status = "cancelled"
		log.Printf("❌ Execution %s cancelled", executionID)
	}

	return nil
}

// CleanupExecutions removes completed executions from memory
func (o *TestOrchestrator) CleanupExecutions() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for id, execution := range o.activeExecutions {
		if execution.EndTime != nil && time.Since(*execution.EndTime) > 1*time.Hour {
			delete(o.activeExecutions, id)
			log.Printf("🧹 Cleaned up execution %s", id)
		}
	}
}
