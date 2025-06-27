# AI Routine Creation System

This directory contains the AI-powered routine creation pipeline for Vrooli. It enables systematic generation, staging, and validation of routine definitions using AI assistance.

## Overview

The AI routine creation system provides a structured workflow for:
- **Ideation**: Collecting routine ideas and requirements in a backlog
- **Generation**: Using AI to convert ideas into complete routine definitions
- **Staging**: Organizing generated routines for review and testing
- **Validation**: Importing and testing routines in the local development environment

## Directory Structure

```
docs/ai-creation/routine/
├── README.md                    # This file
├── prompt.md                    # AI routine generation instructions
├── backlog.md                   # Queue of routine ideas to process
├── subroutine-resolver.md       # Enhanced resolution system documentation
├── staged/                      # Generated routine definitions ready for import
│   ├── subroutines/            # Generated reusable subroutines
│   ├── main-routines/          # Main routines using subroutines
│   └── [legacy files]          # Files from basic generation
├── cache/                       # Resolution system cache
│   ├── search-results.json     # Cached semantic search results
│   ├── staged-index.json       # Index of staged subroutines
│   └── resolution-map.json     # Capability to subroutine mappings
└── templates/                   # Common patterns (future)

scripts/main/
├── routine-generate-enhanced.sh # Smart multi-pass generation (recommended)
├── routine-generate.sh          # Basic generation (legacy)
└── routine-import.sh            # Import and validate staged routines
```

## Quick Start

### 1. Add Routine Ideas
Edit `backlog.md` to add new routine concepts:

```markdown
### Your Routine Name
- **Category**: Productivity
- **Description**: What the routine does and why it's valuable
- **Inputs**: Required input data
- **Outputs**: Expected outputs
- **Strategy**: conversational|reasoning|deterministic
- **Priority**: High|Medium|Low
- **Notes**: Additional context
```

### 2. Generate Routines

#### Option A: Enhanced Generation with Smart Subroutine Resolution (Recommended)
```bash
./scripts/main/routine-generate-enhanced.sh
```

This enhanced system provides:
- **Semantic search** for existing subroutines to avoid duplication
- **Multi-pass generation** that creates missing subroutines automatically  
- **Smart dependency resolution** with proper reuse of components
- **Staged file scanning** to reuse already-generated subroutines
- **Hierarchical organization** with separate subroutines and main routines

#### Option B: Basic Generation (Legacy)
```bash
./scripts/main/routine-generate.sh
```

This calls `maintenance-agent.sh` with the AI creation workflow prompt to automatically read the backlog, generate complete JSON definitions, and save them to `staged/` (may include TODO placeholders for subroutines).

### 3. Import and Test
Import generated routines into your local Vrooli instance:

#### Option A: Using the CLI (Recommended)
```bash
# Ensure local development environment is running
./scripts/main/develop.sh --target docker --detached yes

# Build and install the CLI (first time only)
cd packages/cli
pnpm install
pnpm run build
npm link
cd ../..

# Import routines using CLI
vrooli auth login  # First time only
vrooli routine import-dir ./docs/ai-creation/routine/staged/
```

#### Option B: Using Shell Script (Automated)
```bash
# Ensure local development environment is running
./scripts/main/develop.sh --target docker --detached yes

# Import routines (automatically builds CLI if needed)
./scripts/main/routine-import.sh
```

## Prerequisites

### For Generation (`routine-generate.sh`)
- `maintenance-agent.sh` available in project root
- Properly configured AI model access

### For Import
#### Using CLI (Recommended)
- Local Vrooli development environment running
- API accessible at `http://localhost:5329`
- Node.js and pnpm installed

#### Using Shell Script
- Local Vrooli development environment running
- API accessible at `http://localhost:5329`
- Test user account configured
- `curl` and `jq` installed

## Environment Variables

### For CLI
The CLI stores credentials securely in `~/.vrooli/config.json`. No environment variables needed.

### For Shell Script
Configure these environment variables for the import script:

```bash
export VROOLI_TEST_EMAIL="your-test@example.com"
export VROOLI_TEST_PASSWORD="your-test-password"
```

If not set, defaults to `test@example.com` / `password`.

## CLI vs Shell Script

### Why Use the CLI?

The Vrooli CLI provides several advantages over the shell scripts:

1. **Better Authentication**: Secure token storage with automatic refresh
2. **Progress Tracking**: Real-time progress bars and status updates
3. **Error Handling**: Detailed error messages and recovery
4. **Multi-Environment**: Easy switching between local/staging/production
5. **Validation**: Client-side validation before server calls
6. **Batch Operations**: Efficient handling of multiple routines

### CLI Commands for Routine Management

```bash
# Search for routines using semantic similarity (for finding subroutines)
vrooli routine search "analyze user behavior patterns" --format ids

# Discover all available routines for use as subroutines
vrooli routine discover --format mapping

# Validate routines before import
vrooli routine validate ./staged/my-routine.json

# Import with progress tracking
vrooli routine import-dir ./staged/ --fail-fast

# List imported routines
vrooli routine list --search "ai-generated"

# Export routine for backup
vrooli routine export <routine-id> -o backup.json

# Test routine execution
vrooli routine run <routine-id> --watch
```

## Workflow Details

### Routine Generation Process

1. **AI Workflow**: `routine-generate.sh` calls `maintenance-agent.sh` with a structured prompt
2. **Automatic Processing**: AI reads `backlog.md`, selects first unprocessed item
3. **Dynamic Subroutine Discovery**: 
   - If CLI is authenticated: Automatically discovers all available subroutines and saves to `available-subroutines.txt`
   - AI reads this file to find real subroutine IDs based on functionality needed
   - If unavailable: AI adds TODO comments where manual subroutine selection is needed
4. **Generation**: AI follows `prompt.md` instructions to create complete routine JSON with actual subroutine IDs
5. **Staging**: Generated routine is saved to `staged/` with descriptive filename
6. **Enhanced Validation**: 
   - Structure validation: Checks JSON schema, required fields, ID formats
   - Subroutine validation: Verifies all referenced subroutines exist in the database
   - TODO detection: Flags any placeholder subroutine references
7. **Backlog Management**: AI updates `backlog.md` to mark item as processed

### Import and Validation Process

1. **CLI Auto-Build**: Automatically builds and installs Vrooli CLI if not found
2. **Authentication Check**: Verifies CLI authentication status and prompts login if needed
3. **File Discovery**: Counts and lists JSON files in staged directory
4. **Pre-Import Validation**: Validates all routine JSON files before attempting import
5. **Batch Import**: Imports all valid routines using CLI with fail-fast mode
6. **Success Reporting**: Shows recently imported routines and provides feedback

## Routine Categories

The system supports these routine categories:

- **Metareasoning**: Strategy selection, process optimization, self-reflection
- **Productivity**: Task management, planning, scheduling
- **Knowledge**: Summarization, research, information processing
- **Content**: Writing, communication, creative tasks
- **Analysis**: Data processing, evaluation, insights
- **Integration**: System connections, automation, workflows
- **Learning**: Skill development, feedback, adaptation

## File Formats

### Backlog Items (`backlog.md`)
Simple markdown format with structured metadata for each routine idea.

### Staged Routines (`staged/*.json`)
Complete Vrooli routine definitions following the database schema:

```json
{
  "id": "1234567890123456789",
  "publicId": "routine-name-v1",
  "resourceType": "Routine",
  "versions": [
    {
      "id": "1234567890123456790",
      "resourceSubType": "RoutineMultiStep",
      "config": { /* Complete routine configuration */ },
      "translations": [ /* Name, description, instructions */ ]
    }
  ]
}
```

## Quality Assurance

### Generated Routine Validation
- ✅ Valid JSON structure
- ✅ Required fields present
- ✅ Correct ID formats (19-digit snowflake IDs)
- ✅ Valid subroutine references
- ✅ Proper data flow mapping
- ✅ Complete form definitions

### Runtime Validation
- ✅ Successful API import
- ✅ Database storage
- ✅ Execution smoke tests
- ✅ Error handling verification

## Troubleshooting

### Generation Issues
- **"Could not find maintenance-agent.sh"**: Ensure you're in the project root and the script exists
- **AI generation fails**: Check AI model configuration and token limits
- **Empty staged directory**: Verify backlog has unprocessed items

### Import Issues (CLI)
- **"Vrooli CLI not found"**: Build and install the CLI (see Quick Start section)
- **"Not authenticated"**: Run `vrooli auth login` to authenticate
- **"No response from server"**: Check if API is running on port 5329
- **"Validation failed"**: Run `vrooli routine validate <file>` for detailed errors
- **"Import failed"**: Check server logs and use `--debug` flag for details

### Import Issues (Shell Script)
- **"API is not responding"**: Start local development environment
- **"Authentication failed"**: Check test user credentials exist in local database
- **"Import failed"**: Review routine JSON structure and validation errors
- **"Routine execution test failed"**: Check subroutine references and data mappings

### Common Fixes
```bash
# Restart development environment
./scripts/main/develop.sh --target docker --detached yes

# Check API health (CLI method)
vrooli auth status

# Check API health (manual method)
curl http://localhost:5329/health

# CLI: Debug authentication issues
vrooli auth login --debug

# CLI: Validate routine before import
vrooli routine validate staged/your-routine.json

# CLI: Test import with dry run
vrooli routine import staged/your-routine.json --dry-run

# Verify test user exists
docker compose exec db psql -U postgres -d vrooli -c "SELECT email FROM users WHERE email = 'test@example.com';"

# Check routine JSON structure
jq . staged/your-routine.json
```

## Contributing

When adding new routine ideas to the backlog:

1. **Be Specific**: Provide clear inputs, outputs, and use cases
2. **Choose Appropriate Strategy**: Match the execution strategy to the routine's purpose
3. **Consider Reusability**: Design routines that can be used in multiple contexts
4. **Follow Naming Conventions**: Use descriptive, lowercase names with hyphens

When modifying the system:

1. **Test End-to-End**: Verify the complete pipeline from backlog to import
2. **Update Documentation**: Keep this README current with any changes
3. **Validate Generated Routines**: Ensure they import and execute successfully
4. **Consider Error Handling**: Add robust error checking and user feedback

## Related Documentation

- [`prompt.md`](./prompt.md) - Complete routine generation instructions
- [`backlog.md`](./backlog.md) - Current routine ideas and requirements
- [`/docs/plans/routine-generation-prompt.md`](../../plans/routine-generation-prompt.md) - Original prompt development
- [`/docs/README.md`](../../README.md) - General documentation index