package distribution

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"scenario-to-desktop-api/distribution/providers"
)

// DefaultService implements the Service interface.
type DefaultService struct {
	repo            Repository
	store           Store
	cancelManager   CancelManager
	uploaderFactory UploaderFactory
	envReader       EnvironmentReader
	timeProvider    TimeProvider
	idGenerator     IDGenerator
	logger          *slog.Logger
}

// ServiceOption configures a DefaultService.
type ServiceOption func(*DefaultService)

// WithRepository sets the repository.
func WithRepository(repo Repository) ServiceOption {
	return func(s *DefaultService) {
		s.repo = repo
	}
}

// WithStore sets the store.
func WithStore(store Store) ServiceOption {
	return func(s *DefaultService) {
		s.store = store
	}
}

// WithCancelManager sets the cancel manager.
func WithCancelManager(cm CancelManager) ServiceOption {
	return func(s *DefaultService) {
		s.cancelManager = cm
	}
}

// WithUploaderFactory sets the uploader factory.
func WithUploaderFactory(factory UploaderFactory) ServiceOption {
	return func(s *DefaultService) {
		s.uploaderFactory = factory
	}
}

// WithEnvironmentReader sets the environment reader.
func WithEnvironmentReader(envReader EnvironmentReader) ServiceOption {
	return func(s *DefaultService) {
		s.envReader = envReader
	}
}

// WithTimeProvider sets the time provider.
func WithTimeProvider(tp TimeProvider) ServiceOption {
	return func(s *DefaultService) {
		s.timeProvider = tp
	}
}

// WithIDGenerator sets the ID generator.
func WithIDGenerator(gen IDGenerator) ServiceOption {
	return func(s *DefaultService) {
		s.idGenerator = gen
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) ServiceOption {
	return func(s *DefaultService) {
		s.logger = logger
	}
}

// NewService creates a new distribution service.
func NewService(opts ...ServiceOption) *DefaultService {
	s := &DefaultService{
		envReader:    &RealEnvironmentReader{},
		timeProvider: &RealTimeProvider{},
		idGenerator:  &UUIDGenerator{},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Set defaults for required dependencies
	if s.store == nil {
		s.store = NewInMemoryStore()
	}
	if s.cancelManager == nil {
		s.cancelManager = NewCancelManager()
	}
	if s.uploaderFactory == nil {
		s.uploaderFactory = &DefaultUploaderFactory{}
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}

	return s
}

// Distribute starts a distribution operation.
func (s *DefaultService) Distribute(ctx context.Context, req *DistributeRequest) (*DistributeResponse, error) {
	// Get distribution config
	config, err := s.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get distribution config: %w", err)
	}

	// Determine which targets to use
	targets := s.getTargets(config, req.TargetNames)
	if len(targets) == 0 {
		return nil, fmt.Errorf("no enabled distribution targets found")
	}

	// Validate artifacts exist
	if len(req.Artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts to distribute")
	}

	// Generate distribution ID
	distributionID := s.idGenerator.Generate()

	// Create initial status
	status := &DistributionStatus{
		DistributionID: distributionID,
		ScenarioName:   req.ScenarioName,
		Version:        req.Version,
		Status:         StatusPending,
		StartedAt:      s.timeProvider.NowUnix(),
		Targets:        make(map[string]*TargetDistribution),
	}

	// Initialize target statuses
	for _, target := range targets {
		targetDist := &TargetDistribution{
			TargetName: target.Name,
			Status:     StatusPending,
			Uploads:    make(map[string]*PlatformUpload),
		}
		for platform, localPath := range req.Artifacts {
			targetDist.Uploads[platform] = &PlatformUpload{
				Platform:  platform,
				Status:    StatusPending,
				LocalPath: localPath,
			}
		}
		status.Targets[target.Name] = targetDist
	}

	// Save initial status
	s.store.Save(status)

	// Create cancellable context
	distCtx, cancel := context.WithCancel(ctx)
	s.cancelManager.Set(distributionID, cancel)

	// Start distribution in background
	go s.runDistribution(distCtx, distributionID, req, targets)

	return &DistributeResponse{
		DistributionID: distributionID,
		Status:         StatusRunning,
		StatusURL:      fmt.Sprintf("/api/v1/distribution/status/%s", distributionID),
	}, nil
}

// getTargets returns the targets to use based on request.
func (s *DefaultService) getTargets(config *DistributionConfig, targetNames []string) []*DistributionTarget {
	var targets []*DistributionTarget

	if len(targetNames) == 0 {
		// Use all enabled targets
		for _, target := range config.Targets {
			if target.Enabled {
				targets = append(targets, target)
			}
		}
	} else {
		// Use specified targets
		for _, name := range targetNames {
			if target, ok := config.Targets[name]; ok && target.Enabled {
				targets = append(targets, target)
			}
		}
	}

	return targets
}

// runDistribution executes the distribution operation.
func (s *DefaultService) runDistribution(ctx context.Context, distributionID string, req *DistributeRequest, targets []*DistributionTarget) {
	defer s.cancelManager.Delete(distributionID)

	// Update status to running
	s.store.Update(distributionID, func(status *DistributionStatus) {
		status.Status = StatusRunning
	})

	// Track results
	var mu sync.Mutex
	completedTargets := 0
	failedTargets := 0

	// Use WaitGroup for parallel uploads
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(target *DistributionTarget) {
			defer wg.Done()

			// Check for cancellation
			select {
			case <-ctx.Done():
				s.store.Update(distributionID, func(status *DistributionStatus) {
					if td, ok := status.Targets[target.Name]; ok {
						td.Status = StatusCancelled
						td.Error = "distribution cancelled"
					}
				})
				mu.Lock()
				failedTargets++
				mu.Unlock()
				return
			default:
			}

			// Create uploader for target
			uploader, err := s.uploaderFactory.Create(target, s.envReader)
			if err != nil {
				s.logger.Error("failed to create uploader",
					"target", target.Name,
					"error", err)
				s.store.Update(distributionID, func(status *DistributionStatus) {
					if td, ok := status.Targets[target.Name]; ok {
						td.Status = StatusFailed
						td.Error = fmt.Sprintf("create uploader: %v", err)
					}
				})
				mu.Lock()
				failedTargets++
				mu.Unlock()
				return
			}

			// Upload all artifacts for this target
			success := s.uploadToTarget(ctx, distributionID, target, uploader, req)

			mu.Lock()
			if success {
				completedTargets++
			} else {
				failedTargets++
			}
			mu.Unlock()
		}(target)
	}

	// Wait for all uploads to complete
	wg.Wait()

	// Update final status
	s.store.Update(distributionID, func(status *DistributionStatus) {
		status.CompletedAt = s.timeProvider.NowUnix()

		if failedTargets == 0 {
			status.Status = StatusCompleted
		} else if completedTargets > 0 {
			status.Status = StatusPartial
			status.Error = fmt.Sprintf("%d of %d targets failed", failedTargets, len(targets))
		} else {
			status.Status = StatusFailed
			status.Error = "all targets failed"
		}
	})

	s.logger.Info("distribution completed",
		"distribution_id", distributionID,
		"completed_targets", completedTargets,
		"failed_targets", failedTargets)
}

// uploadToTarget uploads all artifacts to a single target.
func (s *DefaultService) uploadToTarget(ctx context.Context, distributionID string, target *DistributionTarget, uploader Uploader, req *DistributeRequest) bool {
	// Update target status
	s.store.Update(distributionID, func(status *DistributionStatus) {
		if td, ok := status.Targets[target.Name]; ok {
			td.Status = StatusUploading
			td.StartedAt = s.timeProvider.NowUnix()
		}
	})

	allSuccess := true

	for platform, localPath := range req.Artifacts {
		// Check for cancellation
		select {
		case <-ctx.Done():
			s.store.Update(distributionID, func(status *DistributionStatus) {
				if td, ok := status.Targets[target.Name]; ok {
					if pu, ok := td.Uploads[platform]; ok {
						pu.Status = StatusCancelled
						pu.Error = "distribution cancelled"
					}
				}
			})
			return false
		default:
		}

		// Update upload status
		s.store.Update(distributionID, func(status *DistributionStatus) {
			if td, ok := status.Targets[target.Name]; ok {
				if pu, ok := td.Uploads[platform]; ok {
					pu.Status = StatusUploading
				}
			}
		})

		// Build object key
		filename := filepath.Base(localPath)
		key := s.buildObjectKey(req.ScenarioName, req.Version, platform, filename)

		// Create upload request
		uploadReq := &UploadRequest{
			LocalPath: localPath,
			Key:       key,
			ProgressCallback: func(bytesUploaded, totalBytes int64) {
				s.store.Update(distributionID, func(status *DistributionStatus) {
					if td, ok := status.Targets[target.Name]; ok {
						if pu, ok := td.Uploads[platform]; ok {
							pu.BytesUploaded = bytesUploaded
							pu.Size = totalBytes
						}
					}
				})
			},
		}

		// Execute upload
		result, err := uploader.Upload(ctx, uploadReq)
		if err != nil {
			s.logger.Error("upload failed",
				"target", target.Name,
				"platform", platform,
				"error", err)
			s.store.Update(distributionID, func(status *DistributionStatus) {
				if td, ok := status.Targets[target.Name]; ok {
					if pu, ok := td.Uploads[platform]; ok {
						pu.Status = StatusFailed
						pu.Error = err.Error()
					}
				}
			})
			allSuccess = false
			continue
		}

		// Update success status
		s.store.Update(distributionID, func(status *DistributionStatus) {
			if td, ok := status.Targets[target.Name]; ok {
				if pu, ok := td.Uploads[platform]; ok {
					pu.Status = StatusCompleted
					pu.RemoteKey = result.Key
					pu.URL = result.URL
					pu.Size = result.Size
					pu.BytesUploaded = result.Size
				}
			}
		})

		s.logger.Info("upload completed",
			"target", target.Name,
			"platform", platform,
			"key", result.Key,
			"size", result.Size)
	}

	// Update target completion status
	s.store.Update(distributionID, func(status *DistributionStatus) {
		if td, ok := status.Targets[target.Name]; ok {
			td.CompletedAt = s.timeProvider.NowUnix()
			if allSuccess {
				td.Status = StatusCompleted
			} else {
				td.Status = StatusPartial
			}
		}
	})

	return allSuccess
}

// buildObjectKey constructs the S3 object key for an artifact.
func (s *DefaultService) buildObjectKey(scenarioName, version, platform, filename string) string {
	// Format: {scenario}/{version}/{platform}/{filename}
	// Example: my-app/v1.0.0/win/MyApp-Setup-1.0.0.msi
	if version == "" {
		version = "latest"
	}
	return fmt.Sprintf("%s/%s/%s/%s", scenarioName, version, platform, filename)
}

// GetDistributionStatus retrieves the status of a distribution operation.
func (s *DefaultService) GetDistributionStatus(distributionID string) (*DistributionStatus, bool) {
	return s.store.Get(distributionID)
}

// ListDistributions returns all tracked distribution operations.
func (s *DefaultService) ListDistributions() []*DistributionStatus {
	return s.store.List()
}

// CancelDistribution cancels an in-progress distribution.
func (s *DefaultService) CancelDistribution(distributionID string) bool {
	cancel, ok := s.cancelManager.Get(distributionID)
	if !ok {
		return false
	}

	cancel()
	s.cancelManager.Delete(distributionID)

	s.store.Update(distributionID, func(status *DistributionStatus) {
		status.Status = StatusCancelled
		status.CompletedAt = s.timeProvider.NowUnix()
		status.Error = "cancelled by user"
	})

	return true
}

// ValidateTargets validates all (or specified) targets.
func (s *DefaultService) ValidateTargets(ctx context.Context, targetNames []string) *ValidationResult {
	config, err := s.repo.Get(ctx)
	if err != nil {
		return &ValidationResult{
			Valid:   false,
			Targets: map[string]*TargetValidation{},
		}
	}

	targets := s.getTargets(config, targetNames)
	result := &ValidationResult{
		Valid:   true,
		Targets: make(map[string]*TargetValidation),
	}

	for _, target := range targets {
		validation := s.validateTarget(ctx, target)
		result.Targets[target.Name] = validation
		if !validation.Valid {
			result.Valid = false
		}
	}

	return result
}

// validateTarget validates a single target.
func (s *DefaultService) validateTarget(ctx context.Context, target *DistributionTarget) *TargetValidation {
	validation := &TargetValidation{
		TargetName: target.Name,
		Valid:      true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Structural validation
	structErrors := providers.ValidateTarget(targetToProviderTarget(target))
	for _, err := range structErrors {
		if err.Severity == "error" {
			validation.Valid = false
			validation.Errors = append(validation.Errors, err.Message)
		} else {
			validation.Warnings = append(validation.Warnings, err.Message)
		}
	}

	if !validation.Valid {
		return validation
	}

	// Connection validation
	uploader, err := s.uploaderFactory.Create(target, s.envReader)
	if err != nil {
		validation.Valid = false
		validation.Connected = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("create uploader: %v", err))
		return validation
	}

	err = uploader.ValidateCredentials(ctx)
	if err != nil {
		validation.Valid = false
		validation.Connected = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("connection failed: %v", err))
		return validation
	}

	validation.Connected = true
	validation.Permissions = true // If HeadBucket succeeds, we have at least read access

	return validation
}
