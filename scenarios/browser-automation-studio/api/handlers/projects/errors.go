package projects

import "errors"

var (
	errInvalidProjectID         = errors.New("invalid project id")
	errInvalidWorkflowID        = errors.New("invalid workflow id")
	errProjectNotFound          = errors.New("project not found")
	errProjectAlreadyExists     = errors.New("project already exists")
	errProjectFolderTaken       = errors.New("project folder path already in use")
	errNameRequired             = errors.New("name is required")
	errFolderPathRequired       = errors.New("folder_path is required")
	errWorkflowIDsRequired      = errors.New("workflow_ids must not be empty")
	errBulkOperationTooLarge    = errors.New("workflow_ids exceeds maximum bulk size")
	errUnknownPreset            = errors.New("unknown preset")
	errInvalidPresetPath        = errors.New("invalid preset path")
	errProjectPresetFolderError = errors.New("failed to create preset folder")
)
