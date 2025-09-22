package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	maxIssuesPerAutomationLoop = 200
	maxConcurrentAutomatedJobs = 1
)

type fixQueue struct {
	ids    []string
	cursor int
}

func newFixQueue(ids []string) *fixQueue {
	return &fixQueue{ids: append([]string(nil), ids...)}
}

func (q *fixQueue) Reset(ids []string) {
	q.ids = append([]string(nil), ids...)
	q.cursor = 0
}

func (q *fixQueue) Remaining() int {
	if q.cursor >= len(q.ids) {
		return 0
	}
	return len(q.ids) - q.cursor
}

func (q *fixQueue) Next(maxPerLoop, allowance int) []string {
	if q.cursor >= len(q.ids) {
		return nil
	}
	limit := maxPerLoop
	if limit <= 0 {
		limit = len(q.ids) - q.cursor
	}
	if allowance > 0 && allowance < limit {
		limit = allowance
	}
	end := q.cursor + limit
	if end > len(q.ids) {
		end = len(q.ids)
	}
	if end <= q.cursor {
		return nil
	}
	batch := append([]string(nil), q.ids[q.cursor:end]...)
	q.cursor = end
	return batch
}

type AutomatedFixJobOptions struct {
	Scenario         string
	ActiveTypes      []string
	ActiveSeverities []string
	Strategy         string
	LoopDelaySeconds int
	TimeoutSeconds   int
	MaxFixes         int
	Model            string
}

type AutomatedFixJobSnapshot struct {
	ID               string                 `json:"id"`
	Scenario         string                 `json:"scenario"`
	Status           string                 `json:"status"`
	Strategy         string                 `json:"strategy"`
	ActiveTypes      []string               `json:"active_types"`
	ActiveSeverities []string               `json:"active_severities"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	LoopsCompleted   int                    `json:"loops_completed"`
	IssuesAttempted  int                    `json:"issues_attempted"`
	MaxFixes         int                    `json:"max_fixes"`
	Message          string                 `json:"message"`
	Error            string                 `json:"error,omitempty"`
	AutomationRunID  string                 `json:"automation_run_id"`
	Loops            []automationLoopRecord `json:"loops"`
	Model            string                 `json:"model"`
}

type AutomatedFixRunner struct {
	mu    sync.RWMutex
	jobs  map[string]*automatedFixJob
	slots chan struct{}
}

func newAutomatedFixRunner() *AutomatedFixRunner {
	return &AutomatedFixRunner{
		jobs:  make(map[string]*automatedFixJob),
		slots: make(chan struct{}, maxConcurrentAutomatedJobs),
	}
}

func (r *AutomatedFixRunner) Start(options AutomatedFixJobOptions) (*automatedFixJob, error) {
	options.Scenario = strings.TrimSpace(options.Scenario)
	if options.Scenario == "" {
		return nil, errors.New("scenario is required")
	}

	job, err := newAutomatedFixJob(options)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	for _, existing := range r.jobs {
		if existing.scenario == options.Scenario && !existing.isTerminal() {
			r.mu.Unlock()
			return nil, fmt.Errorf("an automated fix job is already running for %s", options.Scenario)
		}
	}
	r.jobs[job.id] = job
	r.mu.Unlock()

	go r.executeJob(job)

	return job, nil
}

func (r *AutomatedFixRunner) executeJob(job *automatedFixJob) {
	defer func() {
		r.mu.Lock()
		delete(r.jobs, job.id)
		r.mu.Unlock()
	}()

	acquired := r.acquireSlot(job)
	if !acquired {
		return
	}
	defer func() {
		<-r.slots
	}()

	job.run()
}

func (r *AutomatedFixRunner) acquireSlot(job *automatedFixJob) bool {
	job.markQueued()
	select {
	case r.slots <- struct{}{}:
		return true
	case <-job.stopCh:
		return false
	case <-job.ctx.Done():
		if job.isCancelled() {
			return false
		}
		job.fail(job.ctx.Err())
		return false
	}
}

func (r *AutomatedFixRunner) Get(jobID string) (*AutomatedFixJobSnapshot, bool) {
	r.mu.RLock()
	job, ok := r.jobs[jobID]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	snapshot := job.Snapshot()
	return &snapshot, true
}

func (r *AutomatedFixRunner) List() []AutomatedFixJobSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]AutomatedFixJobSnapshot, 0, len(r.jobs))
	for _, job := range r.jobs {
		result = append(result, job.Snapshot())
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result
}

func (r *AutomatedFixRunner) Cancel(jobID string) error {
	r.mu.RLock()
	job, ok := r.jobs[jobID]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("automation job %s not found", jobID)
	}
	return job.Cancel()
}

func (r *AutomatedFixRunner) RequestStopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, job := range r.jobs {
		job.RequestStopAfterLoop()
	}
}

type automatedFixJob struct {
	id               string
	scenario         string
	strategy         string
	activeTypes      []string
	activeSeverities []string
	loopDelay        time.Duration
	maxFixes         int
	issuesAttempted  int
	loopsCompleted   int
	status           string
	message          string
	errorMessage     string
	startedAt        time.Time
	completedAt      *time.Time
	automationRunID  string
	loopHistory      []automationLoopRecord
	model            string

	securityQueue  *fixQueue
	standardsQueue *fixQueue

	securitySeverity  map[string]string
	standardsSeverity map[string]string

	severityAllow map[string]struct{}
	typeAllow     map[string]struct{}

	ctx    context.Context
	cancel context.CancelFunc

	stopCh        chan struct{}
	stopRequested bool
	stopOnce      sync.Once

	mu sync.RWMutex
}

type automationLoopRecord struct {
	Number              int                      `json:"number"`
	IssuesDispatched    int                      `json:"issues_dispatched"`
	SecurityDispatched  int                      `json:"security_dispatched"`
	StandardsDispatched int                      `json:"standards_dispatched"`
	RescanTriggered     bool                     `json:"rescan_triggered"`
	RescanResults       []automationRescanResult `json:"rescan_results,omitempty"`
	DurationSeconds     float64                  `json:"duration_seconds"`
	CompletedAt         time.Time                `json:"completed_at"`
	Message             string                   `json:"message"`
}

type automationRescanResult struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func newAutomatedFixJob(options AutomatedFixJobOptions) (*automatedFixJob, error) {
	severityAllow := make(map[string]struct{})
	for _, value := range options.ActiveSeverities {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		severityAllow[value] = struct{}{}
	}
	if len(severityAllow) == 0 {
		for _, value := range automatedFixStore.GetConfig().Severities {
			severityAllow[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
		}
	}

	typeAllow := make(map[string]struct{})
	for _, value := range options.ActiveTypes {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "security" || value == "standards" {
			typeAllow[value] = struct{}{}
		}
	}
	if len(typeAllow) == 0 {
		cfg := automatedFixStore.GetConfig()
		for _, value := range cfg.ViolationTypes {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "security" || value == "standards" {
				typeAllow[value] = struct{}{}
			}
		}
	}
	if len(typeAllow) == 0 {
		return nil, errors.New("no violation types are enabled for automation")
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if options.TimeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutSeconds)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	model := sanitizeAutomationModel(options.Model, openRouterModel)

	job := &automatedFixJob{
		id:                fmt.Sprintf("auto-%s", uuid.NewString()),
		scenario:          options.Scenario,
		strategy:          sanitizeStrategy(options.Strategy, defaultAutomatedFixStrategy),
		activeTypes:       copyStringsMapKeys(typeAllow),
		activeSeverities:  copyStringsMapKeys(severityAllow),
		loopDelay:         time.Duration(options.LoopDelaySeconds) * time.Second,
		maxFixes:          options.MaxFixes,
		status:            "pending",
		startedAt:         time.Now().UTC(),
		automationRunID:   uuid.NewString(),
		securityQueue:     newFixQueue(nil),
		standardsQueue:    newFixQueue(nil),
		securitySeverity:  make(map[string]string),
		standardsSeverity: make(map[string]string),
		severityAllow:     severityAllow,
		typeAllow:         typeAllow,
		ctx:               ctx,
		cancel:            cancel,
		model:             model,
		stopCh:            make(chan struct{}),
	}

	if err := job.refreshQueues(); err != nil {
		return nil, err
	}

	if job.securityQueue.Remaining() == 0 && job.standardsQueue.Remaining() == 0 {
		return nil, errors.New("no matching violations found for automated fixes")
	}

	return job, nil
}

var automatedFixRunner = newAutomatedFixRunner()

func copyStringsMapKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (job *automatedFixJob) Cancel() error {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.status == "completed" || job.status == "failed" || job.status == "cancelled" {
		return nil
	}
	job.status = "cancelled"
	job.message = "Automation cancelled"
	now := time.Now().UTC()
	job.completedAt = &now
	if job.cancel != nil {
		job.cancel()
	}
	job.stopOnce.Do(func() {
		close(job.stopCh)
	})
	return nil
}

func (job *automatedFixJob) isTerminal() bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.status == "completed" || job.status == "failed" || job.status == "cancelled"
}

func (job *automatedFixJob) RequestStopAfterLoop() {
	job.mu.Lock()
	if job.stopRequested || job.status == "completed" || job.status == "failed" || job.status == "cancelled" {
		job.mu.Unlock()
		return
	}
	job.stopRequested = true
	job.mu.Unlock()
	job.stopOnce.Do(func() {
		close(job.stopCh)
	})
}

func (job *automatedFixJob) shouldStop() bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.stopRequested
}

func (job *automatedFixJob) Snapshot() AutomatedFixJobSnapshot {
	job.mu.RLock()
	defer job.mu.RUnlock()
	copy := AutomatedFixJobSnapshot{
		ID:               job.id,
		Scenario:         job.scenario,
		Status:           job.status,
		Strategy:         job.strategy,
		ActiveTypes:      append([]string(nil), job.activeTypes...),
		ActiveSeverities: append([]string(nil), job.activeSeverities...),
		StartedAt:        job.startedAt,
		CompletedAt:      job.completedAt,
		LoopsCompleted:   job.loopsCompleted,
		IssuesAttempted:  job.issuesAttempted,
		MaxFixes:         job.maxFixes,
		Message:          job.message,
		Error:            job.errorMessage,
		AutomationRunID:  job.automationRunID,
		Model:            job.model,
	}
	copy.Loops = cloneLoopHistory(job.loopHistory)
	return copy
}

func (job *automatedFixJob) markQueued() {
	job.mu.Lock()
	if job.status == "pending" {
		job.message = "Waiting for available automation slot"
	}
	job.mu.Unlock()
}

func (job *automatedFixJob) isCancelled() bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.status == "cancelled"
}

func cloneLoopHistory(records []automationLoopRecord) []automationLoopRecord {
	if len(records) == 0 {
		return nil
	}
	cloned := make([]automationLoopRecord, len(records))
	for i, record := range records {
		cloned[i] = record
		if len(record.RescanResults) > 0 {
			cloned[i].RescanResults = make([]automationRescanResult, len(record.RescanResults))
			copy(cloned[i].RescanResults, record.RescanResults)
		}
	}
	return cloned
}

func (job *automatedFixJob) refreshQueues() error {
	if _, ok := job.typeAllow["security"]; ok {
		ids, severityMap, err := job.collectSecurityIssues()
		if err != nil {
			return err
		}
		job.securitySeverity = severityMap
		job.securityQueue.Reset(ids)
	} else {
		job.securityQueue.Reset(nil)
	}

	if _, ok := job.typeAllow["standards"]; ok {
		ids, severityMap, err := job.collectStandardsIssues()
		if err != nil {
			return err
		}
		job.standardsSeverity = severityMap
		job.standardsQueue.Reset(ids)
	} else {
		job.standardsQueue.Reset(nil)
	}

	return nil
}

func (job *automatedFixJob) collectSecurityIssues() ([]string, map[string]string, error) {
	allowed := job.severityAllow
	vulnerabilities := filterVulnerabilitiesBySeverity(job.scenario, allowed)
	if len(vulnerabilities) == 0 {
		return nil, map[string]string{}, nil
	}
	sort.SliceStable(vulnerabilities, func(i, j int) bool {
		return compareSeverities(vulnerabilities[i].Severity, vulnerabilities[j].Severity, job.strategy) < 0
	})
	severities := make(map[string]string, len(vulnerabilities))
	ordered := make([]string, 0, len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		id := strings.TrimSpace(vuln.ID)
		if id == "" {
			continue
		}
		if _, exists := severities[id]; exists {
			continue
		}
		severities[id] = strings.ToLower(strings.TrimSpace(vuln.Severity))
		ordered = append(ordered, id)
	}
	return ordered, severities, nil
}

func (job *automatedFixJob) collectStandardsIssues() ([]string, map[string]string, error) {
	allowed := job.severityAllow
	violations := filterStandardsBySeverity(job.scenario, allowed)
	if len(violations) == 0 {
		return nil, map[string]string{}, nil
	}
	sort.SliceStable(violations, func(i, j int) bool {
		return compareSeverities(violations[i].Severity, violations[j].Severity, job.strategy) < 0
	})
	severities := make(map[string]string, len(violations))
	ordered := make([]string, 0, len(violations))
	for _, violation := range violations {
		id := strings.TrimSpace(violation.ID)
		if id == "" {
			continue
		}
		if _, exists := severities[id]; exists {
			continue
		}
		severities[id] = strings.ToLower(strings.TrimSpace(violation.Severity))
		ordered = append(ordered, id)
	}
	return ordered, severities, nil
}

func compareSeverities(a, b, strategy string) int {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	order := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
	av, aok := order[a]
	bv, bok := order[b]
	if !aok {
		av = len(order)
	}
	if !bok {
		bv = len(order)
	}
	if strategy == "low_first" {
		if av == bv {
			return 0
		}
		return compareInts(bv, av)
	}
	if av == bv {
		return 0
	}
	return compareInts(av, bv)
}

func compareInts(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (job *automatedFixJob) run() {
	job.mu.Lock()
	job.status = "running"
	job.message = "Automation job started"
	job.mu.Unlock()

	defer func() {
		if job.cancel != nil {
			job.cancel()
		}
	}()

	for {
		if job.shouldStop() {
			job.complete("Automation disabled by operator")
			return
		}

		if err := job.ctx.Err(); err != nil {
			job.fail(err)
			return
		}

		shouldContinue, err := job.runLoop()
		if err != nil {
			job.fail(err)
			return
		}

		if job.maxFixes > 0 && job.issuesAttempted >= job.maxFixes {
			job.complete("Maximum automated fixes reached for this job")
			return
		}

		if !shouldContinue {
			job.complete("Automation finished - no remaining issues")
			return
		}

		if job.shouldStop() {
			job.complete("Automation disabled by operator")
			return
		}

		if job.loopDelay > 0 {
			select {
			case <-time.After(job.loopDelay):
			case <-job.ctx.Done():
				job.fail(job.ctx.Err())
				return
			case <-job.stopCh:
				job.complete("Automation disabled by operator")
				return
			}
		}
	}
}

func (job *automatedFixJob) runLoop() (bool, error) {
	loopNumber := job.nextLoopNumber()
	loopStart := time.Now()
	standardsFirst := job.strategy == "standards_first"
	order := []string{"security", "standards"}
	if standardsFirst {
		order = []string{"standards", "security"}
	}

	allowance := 0
	if job.maxFixes > 0 {
		allowance = job.maxFixes - job.issuesAttempted
		if allowance <= 0 {
			return false, nil
		}
	}

	totalLaunched := 0
	securityDispatched := 0
	standardsDispatched := 0
	launchedAgents := make([]string, 0)

	for _, fixType := range order {
		if _, ok := job.typeAllow[fixType]; !ok {
			continue
		}
		queue := job.queueForType(fixType)
		if job.maxFixes > 0 && allowance <= 0 {
			break
		}
		batch := queue.Next(maxIssuesPerAutomationLoop, allowance)
		if len(batch) == 0 {
			continue
		}
		if allowance > 0 {
			allowance -= len(batch)
		}

		agents, err := job.launchBatch(fixType, batch, loopNumber)
		if err != nil {
			return false, err
		}
		for _, agent := range agents {
			launchedAgents = append(launchedAgents, agent.ID)
		}
		totalLaunched += len(batch)
		if fixType == "security" {
			securityDispatched += len(batch)
		} else {
			standardsDispatched += len(batch)
		}
	}

	if totalLaunched > 0 {
		if err := job.waitForAgents(launchedAgents); err != nil {
			return false, err
		}
		job.mu.Lock()
		job.issuesAttempted += totalLaunched
		job.mu.Unlock()
	}

	needRepeat, rescanResults, err := job.handleDepletion(totalLaunched == 0)
	if err != nil {
		return false, err
	}

	loopRecord := automationLoopRecord{
		Number:              loopNumber,
		IssuesDispatched:    totalLaunched,
		SecurityDispatched:  securityDispatched,
		StandardsDispatched: standardsDispatched,
		RescanTriggered:     len(rescanResults) > 0,
		RescanResults:       append([]automationRescanResult(nil), rescanResults...),
		DurationSeconds:     time.Since(loopStart).Seconds(),
		CompletedAt:         time.Now().UTC(),
	}

	switch {
	case totalLaunched > 0 && len(rescanResults) > 0:
		loopRecord.Message = fmt.Sprintf("Dispatched %d issues; triggered rescan", totalLaunched)
	case totalLaunched > 0:
		loopRecord.Message = fmt.Sprintf("Dispatched %d issues", totalLaunched)
	case len(rescanResults) > 0:
		loopRecord.Message = "Queues empty; rescan triggered"
	default:
		loopRecord.Message = "No issues matched automation policy"
	}

	job.mu.Lock()
	job.loopHistory = append(job.loopHistory, loopRecord)
	job.loopsCompleted++
	job.message = loopRecord.Message
	job.mu.Unlock()

	return needRepeat, nil
}

func (job *automatedFixJob) queueForType(fixType string) *fixQueue {
	if fixType == "security" {
		return job.securityQueue
	}
	return job.standardsQueue
}

func (job *automatedFixJob) nextLoopNumber() int {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.loopsCompleted + 1
}

func (job *automatedFixJob) launchBatch(fixType string, batch []string, loopNumber int) ([]*AgentInfo, error) {
	if len(batch) == 0 {
		return nil, nil
	}
	selectedModel := job.model
	metadata := map[string]string{
		"automated_fix":     "true",
		"trigger":           "automation_loop",
		"automation_run_id": job.automationRunID,
		"automation_loop":   strconv.Itoa(loopNumber),
		"strategy":          job.strategy,
		"violation_type":    fixType,
		"violation_count":   strconv.Itoa(len(batch)),
	}
	if selectedModel != "" {
		metadata["model"] = selectedModel
	}

	scenarioTargets := []claudeFixTarget{{
		Scenario: job.scenario,
		IssueIDs: batch,
	}}

	switch fixType {
	case "security":
		return job.launchSecurityAgents(scenarioTargets, selectedModel, metadata)
	case "standards":
		return job.launchStandardsAgents(scenarioTargets, selectedModel, metadata)
	}
	return nil, fmt.Errorf("unsupported fix type %s", fixType)
}

func (job *automatedFixJob) launchSecurityAgents(targets []claudeFixTarget, model string, metadata map[string]string) ([]*AgentInfo, error) {
	multiTargets, _, err := prepareBulkVulnerabilityTargets(targets)
	if err != nil {
		return nil, err
	}
	totalFindings := countVulnerabilityFindings(multiTargets)
	requestedAgents := clampAgentCount(0, totalFindings)
	groups := splitVulnerabilityTargets(multiTargets, requestedAgents)
	if len(groups) == 0 {
		return nil, errors.New("no vulnerabilities matched for automation batch")
	}
	var agents []*AgentInfo
	for idx, group := range groups {
		prompt, label, promptMetadata, err := buildMultiVulnerabilityFixPrompt(group, "")
		if err != nil {
			return nil, err
		}
		combinedMetadata := cloneStringMap(metadata)
		for key, value := range promptMetadata {
			combinedMetadata[key] = value
		}
		combinedMetadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
		combinedMetadata["issue_kind"] = "vulnerabilities"

		issueIDs := collectVulnerabilityIssueIDs(group)
		agentInfo, err := agentManager.StartAgent(AgentStartConfig{
			Label:    label,
			Name:     fmt.Sprintf("%s (automated)", label),
			Action:   agentActionVulnerabilityFix,
			Scenario: job.scenario,
			IssueIDs: issueIDs,
			Prompt:   prompt,
			Model:    model,
			Metadata: combinedMetadata,
		})
		if err != nil {
			return nil, err
		}
		severityLabel := job.highestSeverityLabel(issueIDs, job.securitySeverity)
		automatedFixStore.RecordStart(job.scenario, "security", severityLabel, agentInfo.ID, len(issueIDs), job.automationRunID)
		agents = append(agents, agentInfo)
	}
	return agents, nil
}

func (job *automatedFixJob) launchStandardsAgents(targets []claudeFixTarget, model string, metadata map[string]string) ([]*AgentInfo, error) {
	multiTargets, _, err := prepareBulkStandardsTargets(targets)
	if err != nil {
		return nil, err
	}
	totalViolations := countStandardsViolations(multiTargets)
	requestedAgents := clampAgentCount(0, totalViolations)
	groups := splitStandardsTargets(multiTargets, requestedAgents)
	if len(groups) == 0 {
		return nil, errors.New("no standards violations matched for automation batch")
	}
	var agents []*AgentInfo
	for idx, group := range groups {
		prompt, label, promptMetadata, err := buildMultiStandardsFixPrompt(group, "")
		if err != nil {
			return nil, err
		}
		combinedMetadata := cloneStringMap(metadata)
		for key, value := range promptMetadata {
			combinedMetadata[key] = value
		}
		combinedMetadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
		combinedMetadata["issue_kind"] = "standards"

		issueIDs := collectStandardsIssueIDs(group)
		agentInfo, err := agentManager.StartAgent(AgentStartConfig{
			Label:    label,
			Name:     fmt.Sprintf("%s (automated)", label),
			Action:   agentActionStandardsFix,
			Scenario: job.scenario,
			IssueIDs: issueIDs,
			Prompt:   prompt,
			Model:    model,
			Metadata: combinedMetadata,
		})
		if err != nil {
			return nil, err
		}
		severityLabel := job.highestSeverityLabel(issueIDs, job.standardsSeverity)
		automatedFixStore.RecordStart(job.scenario, "standards", severityLabel, agentInfo.ID, len(issueIDs), job.automationRunID)
		agents = append(agents, agentInfo)
	}
	return agents, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func (job *automatedFixJob) highestSeverityLabel(ids []string, lookup map[string]string) string {
	severities := make([]string, 0, len(ids))
	for _, id := range ids {
		if severity, ok := lookup[id]; ok {
			severities = append(severities, severity)
		}
	}
	return highestSeverityLabel(severities)
}

func (job *automatedFixJob) waitForAgents(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		pending := 0
		for _, id := range ids {
			if _, ok := agentManager.GetAgent(id); ok {
				pending++
				continue
			}
			if history, ok := agentManager.GetAgentHistory(id); ok {
				if history.Status == agentStatusFailed {
					return fmt.Errorf("agent %s failed: %s", id, history.Error)
				}
			}
		}
		if pending == 0 {
			return nil
		}
		select {
		case <-ticker.C:
		case <-job.ctx.Done():
			return job.ctx.Err()
		}
	}
}

func (job *automatedFixJob) handleDepletion(forceRescan bool) (bool, []automationRescanResult, error) {
	secRemaining := job.securityQueue.Remaining()
	stdRemaining := job.standardsQueue.Remaining()
	needRescan := forceRescan || (secRemaining == 0 && stdRemaining == 0)
	if !needRescan {
		return secRemaining > 0 || stdRemaining > 0, nil, nil
	}

	results, err := job.runRescans()
	if err != nil {
		return false, nil, err
	}
	if err := job.refreshQueues(); err != nil {
		return false, nil, err
	}
	secRemaining = job.securityQueue.Remaining()
	stdRemaining = job.standardsQueue.Remaining()
	return secRemaining > 0 || stdRemaining > 0, results, nil
}

func (job *automatedFixJob) runRescans() ([]automationRescanResult, error) {
	results := make([]automationRescanResult, 0, 2)
	if _, ok := job.typeAllow["security"]; ok {
		res, err := job.runSecurityRescan()
		if err != nil {
			return nil, err
		}
		if res.Type != "" {
			results = append(results, res)
		}
	}
	if _, ok := job.typeAllow["standards"]; ok {
		res, err := job.runStandardsRescan()
		if err != nil {
			return nil, err
		}
		if res.Type != "" {
			results = append(results, res)
		}
	}
	return results, nil
}

func (job *automatedFixJob) runSecurityRescan() (automationRescanResult, error) {
	status, err := securityScanManager.StartScan(job.scenario, securityScanRequest{Type: "full"})
	if err != nil {
		return automationRescanResult{}, err
	}
	finalStatus, err := job.waitForSecurityScan(status.ID)
	if err != nil {
		return automationRescanResult{}, err
	}
	message := "Security scan completed"
	if finalStatus != nil && finalStatus.Result != nil {
		message = fmt.Sprintf("Security scan completed (%d findings)", finalStatus.Result.VulnerabilitiesFound)
	} else if finalStatus != nil && finalStatus.Message != "" {
		message = finalStatus.Message
	}
	return automationRescanResult{Type: "security", Status: "success", Message: message}, nil
}

func (job *automatedFixJob) waitForSecurityScan(jobID string) (*SecurityScanStatus, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-job.ctx.Done():
			return nil, job.ctx.Err()
		case <-ticker.C:
			securityJob, ok := securityScanManager.Get(jobID)
			if !ok {
				return nil, fmt.Errorf("security scan %s not found", jobID)
			}
			status := securityJob.snapshot()
			switch status.Status {
			case "completed":
				final := status
				return &final, nil
			case "failed":
				if status.Error != "" {
					return nil, errors.New(status.Error)
				}
				return nil, errors.New("security scan failed")
			case "cancelled":
				return nil, errors.New("security scan cancelled")
			}
		}
	}
}

func (job *automatedFixJob) runStandardsRescan() (automationRescanResult, error) {
	status, err := standardsScanManager.StartScan(job.scenario, "full", nil)
	if err != nil {
		return automationRescanResult{}, err
	}
	finalStatus, err := job.waitForStandardsScan(status.ID)
	if err != nil {
		return automationRescanResult{}, err
	}
	message := "Standards scan completed"
	if finalStatus != nil && finalStatus.Result != nil {
		message = fmt.Sprintf("Standards scan completed (%d violations)", len(finalStatus.Result.Violations))
	} else if finalStatus != nil && finalStatus.Message != "" {
		message = finalStatus.Message
	}
	return automationRescanResult{Type: "standards", Status: "success", Message: message}, nil
}

func (job *automatedFixJob) waitForStandardsScan(jobID string) (*StandardsScanStatus, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-job.ctx.Done():
			return nil, job.ctx.Err()
		case <-ticker.C:
			standardsJob, ok := standardsScanManager.Get(jobID)
			if !ok {
				return nil, fmt.Errorf("standards scan %s not found", jobID)
			}
			status := standardsJob.snapshot()
			switch status.Status {
			case "completed":
				final := status
				return &final, nil
			case "failed":
				if status.Error != "" {
					return nil, errors.New(status.Error)
				}
				return nil, errors.New("standards scan failed")
			case "cancelled":
				return nil, errors.New("standards scan cancelled")
			}
		}
	}
}

func (job *automatedFixJob) fail(err error) {
	job.mu.Lock()
	defer job.mu.Unlock()
	job.status = "failed"
	job.errorMessage = err.Error()
	job.message = "Automation failed"
	now := time.Now().UTC()
	job.completedAt = &now
}

func (job *automatedFixJob) complete(message string) {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.status == "cancelled" {
		return
	}
	job.status = "completed"
	job.message = message
	now := time.Now().UTC()
	job.completedAt = &now
}
