// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Type aliases for module types
type RequirementModule = types.RequirementModule
type ModuleMetadata = types.ModuleMetadata
