package workflow

// This file previously contained all workflow file synchronization logic.
// It has been refactored into focused modules:
//
//   - workflow_files_utils.go   - Path utilities, string conversions, hashing
//   - workflow_files_reader.go  - Reading workflow files from disk
//   - workflow_files_writer.go  - Writing workflows to disk, listing workflows
//   - workflow_files_sync.go    - Project-level synchronization logic
//
// All functionality remains intact; this refactoring improves maintainability
// by separating concerns and reducing file size.
