# Lifecycle Engine - Modular Architecture

## Overview

The Lifecycle Engine has been refactored into a modular architecture for improved maintainability, testability, and reusability. The original monolithic 734-line script has been broken down into focused, single-responsibility modules.

## Architecture

```
lifecycle/
├── engine.sh           # Main orchestrator (simplified)
├── engine.sh.original  # Original monolithic version (backup)
├── lib/                # Modular components
│   ├── parser.sh       # Argument parsing and validation
│   ├── config.sh       # Configuration loading and management
│   ├── condition.sh    # Condition evaluation logic
│   ├── output.sh       # Output/variable management
│   ├── executor.sh     # Step execution engine
│   ├── parallel.sh     # Parallel execution management
│   ├── targets.sh      # Target configuration and inheritance
│   └── phase.sh        # Phase orchestration
```

## Modules

### parser.sh
- Handles command-line argument parsing
- Validates input parameters
- Exports parsed values for other modules
- **Functions**: `parser::parse_args`, `parser::validate`, `parser::export`

### config.sh
- Loads and validates service.json files
- Extracts lifecycle configuration
- Provides configuration access methods
- **Functions**: `config::load_file`, `config::get_phase`, `config::list_phases`

### condition.sh
- Evaluates conditional expressions
- Supports multiple condition types (equals, exists, command, etc.)
- Handles JSON and string condition formats
- **Functions**: `condition::evaluate`, `condition::should_skip_step`

### output.sh
- Manages step outputs and variables
- Handles inter-step communication
- Captures and processes command output
- **Functions**: `output::capture`, `output::set`, `output::get`

### executor.sh
- Executes individual steps
- Handles retry logic and timeouts
- Manages error strategies
- **Functions**: `executor::run_step`, `executor::retry_step`

### parallel.sh
- Manages parallel step execution
- Implements concurrency control
- Supports different parallel strategies (all, race, fail-fast)
- **Functions**: `parallel::execute`, `parallel::execute_group`

### targets.sh
- Handles target-specific configurations
- Resolves target inheritance chains
- Merges target configurations
- **Functions**: `targets::get_config`, `targets::resolve_inheritance`

### phase.sh
- Orchestrates lifecycle phase execution
- Manages phase requirements and confirmations
- Executes hooks and steps in order
- **Functions**: `phase::execute`, `phase::execute_steps`

## Usage

The interface remains the same as the original engine:

```bash
# Basic usage
./engine.sh <phase> [options]

# Examples
./engine.sh setup --target native-linux
./engine.sh develop --target docker --detached yes
./engine.sh build --target k8s --version 2.0.0
./engine.sh test --skip integration-tests
./engine.sh deploy --target k8s --dry-run

# Special commands
./engine.sh --help      # Show help
./engine.sh --version   # Show version
./engine.sh list        # List available phases
```

## Testing

Each module can be tested independently:

```bash
# Test a specific module by sourcing it
source lib/parser.sh
parser::parse_args setup --target docker --dry-run
parser::validate
```

## Benefits of Modular Architecture

1. **Maintainability**: Each module has a single responsibility, making it easier to understand and modify
2. **Testability**: Modules can be tested in isolation with focused test cases
3. **Reusability**: Modules can be sourced and used by other scripts
4. **Debugging**: Issues can be traced to specific modules
5. **Performance**: Only required modules are loaded
6. **Documentation**: Each module is self-documenting with clear interfaces

## Migration from Original

The new modular engine is fully backward compatible. To revert to the original:

```bash
mv engine.sh engine.sh.modular
mv engine.sh.original engine.sh
```

## Module Dependencies

```
engine.sh
├── parser.sh
├── config.sh
├── phase.sh
│   ├── config.sh
│   ├── targets.sh
│   │   └── config.sh
│   ├── parallel.sh
│   │   └── executor.sh
│   │       ├── condition.sh
│   │       └── output.sh
│   └── condition.sh
└── output.sh
```

## Creating New Modules

To add a new module:

1. Create the module in `lib/` directory
2. Follow the naming convention: `module_name.sh`
3. Use function namespaces: `module::function_name`
4. Include error handling and logging
5. Add "source guard" at the end:
   ```bash
   if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
       echo "This script should be sourced, not executed directly" >&2
       exit 1
   fi
   ```
6. Create corresponding tests in `tests/`

## Contributing

When modifying the lifecycle engine:

1. Make changes in the appropriate module
2. Update or add tests for the modified functionality
3. Run all tests to ensure nothing is broken
4. Update this README if adding new modules or changing interfaces