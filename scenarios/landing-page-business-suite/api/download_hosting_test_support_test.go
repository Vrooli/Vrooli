package main

import (
	"strings"

	"landing-page-business-suite-api/internal/delivery"
)

// Test-only aliases keep the established behavior tests focused on observable
// delivery behavior while production code imports the domain package directly.
type (
	DownloadStorage                 = delivery.Storage
	DownloadStorageProvider         = delivery.StorageProvider
	DownloadStorageSettings         = delivery.StorageSettings
	S3DownloadStorageProvider       = delivery.S3StorageProvider
	DownloadStorageSettingsSnapshot = delivery.StorageSettingsSnapshot
	DownloadStorageSettingsUpdate   = delivery.StorageSettingsUpdate
	PresignUploadRequest            = delivery.PresignUploadRequest
	PresignUploadResponse           = delivery.PresignUploadResponse
	CommitArtifactRequest           = delivery.CommitArtifactRequest
	DownloadArtifact                = delivery.Artifact
	ListArtifactsResult             = delivery.ListArtifactsResult
	DownloadHostingService          = delivery.Service
)

var ErrDownloadStorageNotConfigured = delivery.ErrStorageNotConfigured

func NewDownloadHostingService(db delivery.Store, providers ...delivery.StorageProvider) *delivery.Service {
	return delivery.NewService(db, providers...)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
