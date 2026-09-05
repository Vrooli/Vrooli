package validationmatrix

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	store     Store
	executors Executors
	reporter  ReleaseReporter
	catalog   CatalogResolver
	now       func() time.Time
	mu        sync.Mutex
	active    map[string]context.CancelFunc
	waiters   map[string]chan struct{}
}

type ServiceOption func(*Service)

func WithReleaseReporter(reporter ReleaseReporter) ServiceOption {
	return func(s *Service) { s.reporter = reporter }
}

func WithCatalogResolver(catalog CatalogResolver) ServiceOption {
	return func(s *Service) { s.catalog = catalog }
}

func WithClock(now func() time.Time) ServiceOption {
	return func(s *Service) { s.now = now }
}

func NewService(store Store, executors Executors, options ...ServiceOption) *Service {
	s := &Service{store: store, executors: executors, now: time.Now, active: make(map[string]context.CancelFunc), waiters: make(map[string]chan struct{})}
	for _, option := range options {
		option(s)
	}
	return s
}

// Catalog returns the provider-owned catalog snapshot without changing a
// matrix. The UI uses this read-only seam to select existing journeys; matrix
// creation still persists the exact selection supplied by the operator.
func (s *Service) Catalog(ctx context.Context, scenario string) (CatalogSnapshot, error) {
	if s == nil || s.catalog == nil {
		return CatalogSnapshot{}, fmt.Errorf("validation catalog resolver is unavailable")
	}
	return s.catalog.Resolve(ctx, scenario)
}

// RecoverStale marks work whose server owner disappeared as failed before a
// new client can observe it. It never converts missing evidence into a pass.
func (s *Service) RecoverStale() int {
	if s == nil || s.store == nil {
		return 0
	}
	recovered := 0
	for _, run := range s.store.List() {
		if run.State == RunRunning {
			if err := s.MarkStale(run.RunID); err == nil {
				recovered++
			}
		}
	}
	return recovered
}

func (s *Service) Create(selection MatrixSelection) (*MatrixRun, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("validation matrix store is unavailable")
	}
	if s.catalog != nil {
		resolved, err := selection.WithCatalog(context.Background(), s.catalog)
		if err != nil {
			return nil, err
		}
		selection = resolved
	}
	if err := selection.validate(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.store.GetByIdempotencyKey(selection.IdempotencyKey); ok {
		return existing, nil
	}
	now := s.now()
	runID := newID("run")
	matrixID := newID("matrix")
	matrix := &domainv1.ValidationMatrix{
		MatrixId: matrixID, ScenarioName: selection.ScenarioName, ArtifactDigest: selection.ArtifactDigest,
		CreatedAt:           timestamppb.New(now),
		EnvironmentProfiles: append([]domainv1.ValidationEnvironmentProfile(nil), selection.EnvironmentProfiles...),
	}
	run := &MatrixRun{RunID: runID, IdempotencyKey: selection.IdempotencyKey, Matrix: matrix, Selection: cloneSelection(selection), State: RunQueued, CreatedAt: now, UpdatedAt: now}
	for _, journey := range selection.Journeys {
		run.Rows = append(run.Rows, MatrixRow{ID: stableID("row", journey.JourneyID), JourneyID: journey.JourneyID})
	}
	for _, target := range selection.Targets {
		run.Columns = append(run.Columns, MatrixColumn{ID: stableID("column", target.Descriptor.GetTargetId()), TargetID: target.Descriptor.GetTargetId()})
	}
	for _, profile := range selection.EnvironmentProfiles {
		run.Profiles = append(run.Profiles, MatrixProfile{ID: stableID("profile", fmt.Sprint(profile)), Profile: profile})
	}
	for _, journey := range selection.Journeys {
		matrix.Journeys = append(matrix.Journeys, &domainv1.JourneyCatalogItem{JourneyId: journey.JourneyID, DisplayName: journey.DisplayName, SourcePath: journey.SourcePath, ExecutionMode: journey.ExecutionMode, Required: journey.Required, RequiredCapabilities: append([]domainv1.ValidationTargetCapability(nil), journey.RequiredCapabilities...)})
	}
	for _, target := range selection.Targets {
		matrix.Targets = append(matrix.Targets, proto.Clone(target.Descriptor).(*domainv1.ValidationTargetDescriptor))
	}
	for _, journey := range selection.Journeys {
		for _, target := range selection.Targets {
			for _, profile := range selection.EnvironmentProfiles {
				cell := newCell(selection.ScenarioName, selection.ArtifactDigest, journey, target.Descriptor, profile, now)
				run.Cells = append(run.Cells, &CellRecord{Cell: cell, RowID: stableID("row", journey.JourneyID), ColumnID: stableID("column", target.Descriptor.GetTargetId()), ProfileID: stableID("profile", fmt.Sprint(profile)), TargetKind: target.Kind, State: initialCellState(cell), UpdatedAt: now})
				matrix.Cells = append(matrix.Cells, cell)
			}
		}
	}
	if err := s.store.Save(run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) Start(runID string) (*MatrixRun, error) {
	run, ok := s.store.Get(runID)
	if !ok {
		return nil, fmt.Errorf("matrix run %q not found", runID)
	}
	s.mu.Lock()
	if _, running := s.active[runID]; running {
		s.mu.Unlock()
		return run, nil
	}
	if run.State.Terminal() {
		s.mu.Unlock()
		return run, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.active[runID] = cancel
	s.waiters[runID] = make(chan struct{})
	s.mu.Unlock()
	go s.execute(ctx, runID)
	return run, nil
}

func (s *Service) Wait(ctx context.Context, runID string) (*MatrixRun, error) {
	for {
		run, ok := s.store.Get(runID)
		if !ok {
			return nil, fmt.Errorf("matrix run %q not found", runID)
		}
		if run.State.Terminal() {
			return run, nil
		}
		s.mu.Lock()
		waiter := s.waiters[runID]
		s.mu.Unlock()
		if waiter == nil {
			return run, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waiter:
		}
	}
}

func (s *Service) List(scenario string) []*MatrixRun {
	if s == nil || s.store == nil {
		return nil
	}
	all := s.store.List()
	if strings.TrimSpace(scenario) == "" {
		return all
	}
	filtered := make([]*MatrixRun, 0, len(all))
	for _, run := range all {
		if run != nil && run.Selection.ScenarioName == scenario {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

func (s *Service) Compare(currentID, priorID string) (*MatrixComparison, error) {
	current, ok := s.store.Get(currentID)
	if !ok {
		return nil, fmt.Errorf("matrix run %q not found", currentID)
	}
	prior, ok := s.store.Get(priorID)
	if !ok {
		return nil, fmt.Errorf("matrix run %q not found", priorID)
	}
	if current.Selection.ScenarioName != prior.Selection.ScenarioName {
		return nil, fmt.Errorf("matrix runs belong to different scenarios")
	}
	comparison := &MatrixComparison{CurrentRunID: current.RunID, PriorRunID: prior.RunID, ScenarioName: current.Selection.ScenarioName, CurrentArtifactDigest: current.Matrix.GetArtifactDigest(), PriorArtifactDigest: prior.Matrix.GetArtifactDigest()}
	priorCells := make(map[string]*domainv1.ValidationCell, len(prior.Matrix.GetCells()))
	for _, cell := range prior.Matrix.GetCells() {
		if key := cellKey(cell); key != "" {
			priorCells[key] = cell
		}
	}
	for _, cell := range current.Matrix.GetCells() {
		key := cellKey(cell)
		if key == "" {
			continue
		}
		before := priorCells[key]
		item := CellComparison{Key: key, CurrentCellID: cell.GetCellId(), CurrentDisposition: cell.GetDisposition(), CurrentEvidenceCount: len(cell.GetEvidence())}
		if before != nil {
			item.PriorCellID = before.GetCellId()
			item.PriorDisposition = before.GetDisposition()
			item.PriorEvidenceCount = len(before.GetEvidence())
		}
		item.Changed = before == nil || item.CurrentDisposition != item.PriorDisposition || item.CurrentEvidenceCount != item.PriorEvidenceCount
		comparison.Changed = comparison.Changed || item.Changed
		comparison.Cells = append(comparison.Cells, item)
		delete(priorCells, key)
	}
	for key, before := range priorCells {
		comparison.Changed = true
		comparison.Cells = append(comparison.Cells, CellComparison{Key: key, PriorCellID: before.GetCellId(), PriorDisposition: before.GetDisposition(), PriorEvidenceCount: len(before.GetEvidence()), Changed: true})
	}
	return comparison, nil
}

func (s *Service) Abort(runID string) error {
	run, ok := s.store.Get(runID)
	if !ok {
		return fmt.Errorf("matrix run %q not found", runID)
	}
	s.mu.Lock()
	cancel := s.active[runID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		return nil
	}
	if run.State.Terminal() {
		return nil
	}
	now := s.now()
	for _, cell := range run.Cells {
		if cell.State == CellQueued || cell.State == CellRetrying || cell.State == CellRunning {
			cell.State = CellCancelled
			cell.Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
			cell.Cell.Reason = stringPtr("matrix run aborted before cell completion")
			cell.TerminalAt = now
		}
	}
	run.State, run.UpdatedAt, run.CompletedAt = RunCancelled, now, now
	return s.finish(run)
}

func (s *Service) MarkStale(runID string) error {
	run, ok := s.store.Get(runID)
	if !ok {
		return fmt.Errorf("matrix run %q not found", runID)
	}
	if run.State.Terminal() {
		return nil
	}
	now := s.now()
	for _, cell := range run.Cells {
		if cell.State == CellRunning || cell.State == CellRetrying || cell.State == CellQueued {
			cell.State = CellFailed
			cell.Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED
			if cell.Attempts == 0 {
				cell.Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
			}
			cell.Cell.Reason = stringPtr("cell owner became stale before completion")
			cell.TerminalAt = now
		}
	}
	run.State, run.UpdatedAt, run.CompletedAt = RunFailed, now, now
	syncMatrixCells(run)
	run.Gate = EvaluateReleaseGate(run.Matrix)
	return s.finish(run)
}

func (s *Service) Rerun(runID string, selector RerunSelector) (*MatrixRun, error) {
	if !selector.valid() {
		return nil, fmt.Errorf("invalid rerun selector %q", selector.Kind)
	}
	source, ok := s.store.Get(runID)
	if !ok {
		return nil, fmt.Errorf("matrix run %q not found", runID)
	}
	selection := cloneSelection(source.Selection)
	// A rerun is a new durable execution, even when the original request used
	// an idempotency key. Reusing that key would make Create return the source
	// run and mutate its evidence in place.
	selection.IdempotencyKey = ""
	run, err := s.Create(selection)
	if err != nil {
		return nil, err
	}
	run.ParentRunID = source.RunID
	for index := range run.Cells {
		if index >= len(source.Cells) {
			break
		}
		if !selector.matches(source.Cells[index]) {
			run.Cells[index] = cloneCellRecord(source.Cells[index])
			run.Matrix.Cells[index] = proto.Clone(source.Cells[index].Cell).(*domainv1.ValidationCell)
		}
	}
	run.UpdatedAt = s.now()
	if err := s.store.Save(run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) execute(ctx context.Context, runID string) {
	run, ok := s.store.Get(runID)
	if !ok {
		s.finishActive(runID)
		return
	}
	now := s.now()
	run.State, run.UpdatedAt = RunRunning, now
	_ = s.store.Save(run)
	max := run.Selection.MaxConcurrency
	if max <= 0 {
		max = 1
	}
	semaphores := make(map[string]chan struct{})
	var wg sync.WaitGroup
	for _, record := range run.Cells {
		if record.State != CellQueued {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		targetID := record.Cell.GetTargetId()
		semaphore := semaphores[targetID]
		if semaphore == nil {
			semaphore = make(chan struct{}, max)
			semaphores[targetID] = semaphore
		}
		semaphore <- struct{}{}
		wg.Add(1)
		go func(record *CellRecord, semaphore chan struct{}) {
			defer wg.Done()
			defer func() { <-semaphore }()
			s.executeCell(ctx, runID, record)
		}(record, semaphore)
	}
	wg.Wait()
	run, ok = s.store.Get(runID)
	if !ok {
		s.finishActive(runID)
		return
	}
	if ctx.Err() != nil {
		now = s.now()
		for _, record := range run.Cells {
			if record.State == CellQueued || record.State == CellRetrying || record.State == CellRunning {
				record.State = CellCancelled
				record.Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
				record.Cell.Reason = stringPtr("matrix run aborted")
				record.TerminalAt = now
			}
		}
		run.State = RunCancelled
		syncMatrixCells(run)
		run.Gate = EvaluateReleaseGate(run.Matrix)
	} else {
		syncMatrixCells(run)
		run.Gate = EvaluateReleaseGate(run.Matrix)
		if run.Gate.GetPassed() {
			run.State = RunCompleted
		} else {
			run.State = RunFailed
		}
	}
	run.UpdatedAt, run.CompletedAt = s.now(), s.now()
	_ = s.finish(run)
}

func (s *Service) executeCell(ctx context.Context, runID string, record *CellRecord) {
	run, ok := s.store.Get(runID)
	if !ok {
		return
	}
	request := CellRequest{
		RunID:          run.RunID,
		MatrixID:       run.Matrix.GetMatrixId(),
		Command:        run.Selection.Command,
		Args:           append([]string(nil), run.Selection.CommandArgs...),
		ArtifactDigest: run.Matrix.GetArtifactDigest(),
		ArtifactPath:   run.Selection.ArtifactPath,
		Cell:           record.Cell,
		Journey:        journeySelection(run.Selection.Journeys, record.Cell.GetJourneyId()),
		Target:         targetSelection(run.Selection.Targets, record.Cell.GetTargetId()),
		Metadata:       cloneStringMap(run.Selection.Metadata),
	}
	for {
		record.Attempts++
		if !s.updateCell(runID, record, func(current *CellRecord) {
			current.State, current.Attempts, current.StartedAt = CellRunning, record.Attempts, s.now()
		}) {
			return
		}
		var result CellResult
		switch record.TargetKind {
		case TargetLocal:
			result = s.executors.Execute(ctx, TargetLocal, request)
		case TargetBridge:
			result = s.executors.Execute(ctx, TargetBridge, request)
		default:
			result = unavailableResult("target adapter kind is unavailable")
		}
		if ctx.Err() != nil {
			s.updateCell(runID, record, func(current *CellRecord) {
				current.State = CellCancelled
				current.Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
				current.Cell.Reason = stringPtr("cell cancelled with matrix run")
				current.TerminalAt = s.now()
			})
			return
		}
		if result.Retryable && record.Attempts < 2 {
			s.updateCell(runID, record, func(current *CellRecord) {
				current.State = CellRetrying
				current.Cell.Reason = stringPtr(strings.TrimSpace(result.Reason))
			})
			continue
		}
		s.updateCell(runID, record, func(current *CellRecord) {
			current.Cell.Disposition = result.Disposition
			current.Cell.Reason = stringPtr(strings.TrimSpace(result.Reason))
			current.Cell.Evidence = append([]*domainv1.LayeredEvidence(nil), result.Evidence...)
			current.Report = cloneReport(result.Report)
			if result.Disposition == domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS {
				current.State = CellCompleted
			} else {
				current.State = CellFailed
			}
			current.TerminalAt = s.now()
		})
		return
	}
}

func journeySelection(journeys []JourneySelection, id string) JourneySelection {
	for _, journey := range journeys {
		if journey.JourneyID == id {
			return journey
		}
	}
	return JourneySelection{JourneyID: id}
}

func targetSelection(targets []TargetSelection, id string) *domainv1.ValidationTargetDescriptor {
	for _, target := range targets {
		if target.Descriptor != nil && target.Descriptor.GetTargetId() == id {
			return proto.Clone(target.Descriptor).(*domainv1.ValidationTargetDescriptor)
		}
	}
	return nil
}

func (s *Service) updateCell(runID string, record *CellRecord, mutate func(*CellRecord)) bool {
	return s.store.Update(runID, func(run *MatrixRun) {
		for _, current := range run.Cells {
			if current.Cell.GetCellId() == record.Cell.GetCellId() {
				mutate(current)
				current.UpdatedAt = s.now()
				run.UpdatedAt = current.UpdatedAt
				return
			}
		}
	})
}

func (s *Service) finish(run *MatrixRun) error {
	if err := s.store.Save(run); err != nil {
		return err
	}
	if run.Gate != nil && s.reporter != nil {
		var evidence []*domainv1.LayeredEvidence
		for _, cell := range run.Matrix.GetCells() {
			evidence = append(evidence, cell.GetEvidence()...)
		}
		err := s.reporter.ReportValidationGate(context.Background(), ReleaseVerdict{RunID: run.RunID, MatrixID: run.Matrix.GetMatrixId(), ScenarioName: run.Matrix.GetScenarioName(), ArtifactDigest: run.Matrix.GetArtifactDigest(), Gate: run.Gate, Evidence: evidence})
		run.ReleaseReport.ReportedAt = s.now()
		if err != nil {
			run.ReleaseReport.Error = err.Error()
		}
		_ = s.store.Save(run)
	}
	s.finishActive(run.RunID)
	return nil
}

func (s *Service) finishActive(runID string) {
	s.mu.Lock()
	delete(s.active, runID)
	if waiter := s.waiters[runID]; waiter != nil {
		close(waiter)
		delete(s.waiters, runID)
	}
	s.mu.Unlock()
}

func newCell(scenarioName, digest string, journey JourneySelection, target *domainv1.ValidationTargetDescriptor, profile domainv1.ValidationEnvironmentProfile, now time.Time) *domainv1.ValidationCell {
	cell := &domainv1.ValidationCell{CellId: newID("cell"), ScenarioName: scenarioName, ArtifactDigest: digest, TargetId: target.GetTargetId(), JourneyId: journey.JourneyID, EnvironmentProfile: profile, Required: journey.Required, Applicable: true, CreatedAt: timestamppb.New(now), RequiredCapabilities: append([]domainv1.ValidationTargetCapability(nil), journey.RequiredCapabilities...)}
	cell.Applicable, cell.Disposition, cell.Reason = ComputeApplicability(target, journey.RequiredCapabilities, profile)
	return cell
}

// ComputeApplicability is deliberately fail-closed: a selected required cell
// stays applicable when it cannot run, so the release gate records the typed
// failure instead of silently excluding the cell.
func ComputeApplicability(target *domainv1.ValidationTargetDescriptor, required []domainv1.ValidationTargetCapability, profile domainv1.ValidationEnvironmentProfile) (bool, domainv1.ValidationDisposition, *string) {
	if target == nil || !target.GetAvailable() {
		return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, stringPtr("target is unavailable")
	}
	if contract, known := profileContract(profile); known {
		required = append(append([]domainv1.ValidationTargetCapability(nil), required...), contract.RequiredCapabilities...)
		if missing := firstMissingCapability(target.GetCapabilities(), required); missing != domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UNSPECIFIED {
			return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED, stringPtr("target lacks required capability " + missing.String())
		}
		if !contract.Executable {
			return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED, stringPtr("environment profile " + profile.String() + " has no executable adapter")
		}
	} else if profile != domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UNSPECIFIED {
		return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED, stringPtr("environment profile has no target contract")
	}
	if missing := firstMissingCapability(target.GetCapabilities(), required); missing != domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UNSPECIFIED {
		return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED, stringPtr("target lacks required capability " + missing.String())
	}
	if profile == domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UNSPECIFIED {
		return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, stringPtr("environment profile is unspecified")
	}
	return true, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSPECIFIED, nil
}

func initialCellState(cell *domainv1.ValidationCell) CellState {
	if cell.GetDisposition() == domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE || cell.GetDisposition() == domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED {
		return CellFailed
	}
	return CellQueued
}

func firstMissingCapability(have, required []domainv1.ValidationTargetCapability) domainv1.ValidationTargetCapability {
	set := make(map[domainv1.ValidationTargetCapability]struct{}, len(have))
	for _, capability := range have {
		set[capability] = struct{}{}
	}
	for _, capability := range required {
		if _, ok := set[capability]; !ok {
			return capability
		}
	}
	return domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UNSPECIFIED
}

func syncMatrixCells(run *MatrixRun) {
	if run == nil || run.Matrix == nil {
		return
	}
	run.Matrix.Cells = make([]*domainv1.ValidationCell, len(run.Cells))
	for index, record := range run.Cells {
		if record != nil && record.Cell != nil {
			run.Matrix.Cells[index] = proto.Clone(record.Cell).(*domainv1.ValidationCell)
		}
	}
}

func unavailableResult(reason string) CellResult {
	return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: reason}
}

func (s RerunSelector) valid() bool {
	switch strings.ToLower(strings.TrimSpace(s.Kind)) {
	case RerunAll, RerunFailed:
		return true
	case RerunJourney:
		return s.JourneyID != ""
	case RerunRow:
		return s.JourneyID != ""
	case RerunTarget:
		return s.TargetID != ""
	case RerunColumn:
		return s.TargetID != ""
	case RerunCell:
		return s.CellID != ""
	default:
		return false
	}
}

func cloneSelection(selection MatrixSelection) MatrixSelection {
	copy := selection
	copy.CommandArgs = append([]string(nil), selection.CommandArgs...)
	copy.Journeys = append([]JourneySelection(nil), selection.Journeys...)
	copy.Targets = make([]TargetSelection, len(selection.Targets))
	for i, target := range selection.Targets {
		copy.Targets[i] = TargetSelection{Kind: target.Kind}
		if target.Descriptor != nil {
			copy.Targets[i].Descriptor = proto.Clone(target.Descriptor).(*domainv1.ValidationTargetDescriptor)
		}
	}
	copy.EnvironmentProfiles = append([]domainv1.ValidationEnvironmentProfile(nil), selection.EnvironmentProfiles...)
	return copy
}

func cloneCellRecord(record *CellRecord) *CellRecord {
	copy := *record
	copy.Cell = proto.Clone(record.Cell).(*domainv1.ValidationCell)
	copy.Report = cloneReport(record.Report)
	return &copy
}

func cloneReport(report map[string]string) map[string]string {
	if len(report) == 0 {
		return nil
	}
	copy := make(map[string]string, len(report))
	for key, value := range report {
		copy[key] = value
	}
	return copy
}

func newID(prefix string) string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%x", prefix, bytes)
}

func stableID(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%s-%x", prefix, digest[:6])
}
