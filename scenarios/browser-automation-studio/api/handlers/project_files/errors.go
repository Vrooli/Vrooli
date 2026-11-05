package project_files //nolint:revive // Package name mirrors the public project_files proto and REST domain.

import "errors"

var (
	errInvalidProjectID = errors.New("invalid project id")
	errInvalidPath      = errors.New("invalid path")
	errPathEscapesRoot  = errors.New("path escapes project root")
	errProjectNotFound  = errors.New("project not found")
	errFileNotFound     = errors.New("file not found")
	errFileExists       = errors.New("file already exists")
	errOnlyJSONReadable = errors.New("only JSON workflow files are readable")
	errWorkflowFileExt  = errors.New("workflow files must end with .json")
	errEmptyFolderPath  = errors.New("project folder path is empty")
)
