package exports_service

import "errors"

var (
	errInvalidExportID    = errors.New("invalid export id")
	errInvalidExecutionID = errors.New("invalid execution id")
	errInvalidWorkflowID  = errors.New("invalid workflow id")
	errNameRequired       = errors.New("name is required")
	errFormatRequired     = errors.New("format is required")
	errInvalidFormat      = errors.New("format must be one of: mp4, gif, json, html")
	errInvalidStatus      = errors.New("status must be one of: pending, processing, completed, failed")
	errExportNotFound     = errors.New("export not found")
	errExecutionNotFound  = errors.New("execution not found")
	errNoStoragePath      = errors.New("export does not have a storage path")
)
