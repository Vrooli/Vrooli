package artifacts

import sharedfs "test-genie/internal/shared/filesystem"

// FileSystem preserves the artifact package API as the shared write contract.
type FileSystem = sharedfs.Writer

// OSFileSystem preserves the artifact-owner name while sharing the OS adapter.
type OSFileSystem = sharedfs.OS
