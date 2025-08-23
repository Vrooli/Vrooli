# Claude Code Tool Reference

This document provides a comprehensive reference for all tools available in Claude Code, their capabilities, security implications, and usage patterns.

## 🧰 Available Tools

Claude Code has access to the following tools: **Bash, Glob, Grep, LS, ExitPlanMode, Read, Edit, MultiEdit, Write, NotebookRead, NotebookEdit, WebFetch, TodoWrite, WebSearch**.

## 📂 File Operations

### Read
**Purpose**: Read any file with support for partial reading and image viewing  
**Security**: Safe - read-only operation  
**Syntax**: `Read`

**Capabilities**:
- Read up to 2000 lines by default
- Support for line offset and limit parameters
- Can read images (PNG, JPG) and PDFs visually
- Truncates lines longer than 2000 characters
- Returns content in `cat -n` format with line numbers

**Usage Examples**:
```bash
# Safe code analysis
--allowed-tools "Read,Grep"

# Review specific files
--allowed-tools "Read" --prompt "Review src/auth.js for security issues"
```

### Write
**Purpose**: Create new files (requires Read for existing files)  
**Security**: Moderate risk - creates new files  
**Syntax**: `Write`

**Capabilities**:
- Creates new files only
- Must use Read tool first for existing files
- Overwrites existing files if Read was used
- No emoji creation unless explicitly requested

**Usage Examples**:
```bash
# Documentation generation
--allowed-tools "Read,Write" --prompt "Generate API documentation"

# Create new modules
--allowed-tools "Write" --prompt "Create new utility module"
```

### Edit
**Purpose**: Single find-and-replace operation  
**Security**: High risk - modifies existing code  
**Syntax**: `Edit`

**Capabilities**:
- Exact string replacement only
- Must provide unique old_string for matching
- Preserves exact indentation and formatting
- Fails if old_string is not unique (use MultiEdit for multiple)
- Includes replace_all option for renaming variables

**Usage Examples**:
```bash
# Bug fixes
--allowed-tools "Read,Edit" --prompt "Fix the authentication bug"

# Controlled refactoring
--allowed-tools "Edit" --prompt "Rename function getCurrentUser to getActiveUser"
```

### MultiEdit
**Purpose**: Multiple edits to a single file in one atomic operation  
**Security**: High risk - multiple modifications  
**Syntax**: `MultiEdit`

**Capabilities**:
- Atomic operation - all edits succeed or none apply
- Edits applied sequentially in provided order
- Each edit must be valid for operation to succeed
- Built on top of Edit tool with same requirements

**Usage Examples**:
```bash
# Complex refactoring
--allowed-tools "Read,MultiEdit" --prompt "Refactor class to use modern syntax"

# Multiple bug fixes
--allowed-tools "MultiEdit" --prompt "Fix all the validation issues in this file"
```

## 🔍 Search & Discovery

### Grep
**Purpose**: Powerful search tool built on ripgrep  
**Security**: Safe - read-only search  
**Syntax**: `Grep`

**Capabilities**:
- Full regex syntax support
- File filtering with glob patterns
- Multiple output modes: content, files_with_matches, count
- Context lines (-A, -B, -C) for content mode
- Multiline matching support
- Line numbers in output

**Usage Examples**:
```bash
# Security audits
--allowed-tools "Grep,Read" --prompt "Find all SQL queries and check for injection risks"

# Code pattern analysis
--allowed-tools "Grep" --prompt "Find all TODO comments and prioritize them"
```

### Glob
**Purpose**: Fast file pattern matching  
**Security**: Safe - discovery only  
**Syntax**: `Glob`

**Capabilities**:
- Supports glob patterns like `**/*.js` or `src/**/*.ts`
- Returns paths sorted by modification time
- Works with any codebase size
- No file content access - just path matching

**Usage Examples**:
```bash
# Find specific file types
--allowed-tools "Glob,Read" --prompt "Analyze all TypeScript files for type issues"

# Project structure analysis
--allowed-tools "Glob" --prompt "Document the project structure"
```

### LS
**Purpose**: List files and directories  
**Security**: Safe - directory listing only  
**Syntax**: `LS`

**Capabilities**:
- Requires absolute paths (not relative)
- Optional glob pattern ignoring
- Basic directory structure viewing

**Usage Examples**:
```bash
# Project exploration
--allowed-tools "LS,Read" --prompt "Explore and document the project structure"
```

## 🖥️ System Operations

### Bash
**Purpose**: Execute shell commands with persistent session  
**Security**: EXTREMELY HIGH RISK - full system access  
**Syntax**: `Bash` or `Bash(*)` or `Bash(pattern)`

**Capabilities**:
- Persistent shell session with optional timeout (up to 10 minutes)
- Inherits your bash environment and tools
- Proper file path quoting for spaces
- Output truncation after 30,000 characters
- Should NOT use `find`, `grep`, `cat`, `head`, `tail`, `ls` (use specialized tools instead)

**Security Patterns**:
```bash
# Full access (DANGEROUS)
--allowed-tools "Bash(*)"

# Limited to specific commands
--allowed-tools "Bash(git *)" # Git operations only
--allowed-tools "Bash(npm test)" # Specific command only
--allowed-tools "Bash(git commit:*)" # Git commits with any message

# Safe combinations
--allowed-tools "Read,Edit,Bash(git commit:*)" # Code changes + commit
```

**Usage Examples**:
```bash
# Safe testing
--allowed-tools "Bash(npm test),Read" --prompt "Run tests and analyze failures"

# Git operations
--allowed-tools "Edit,Bash(git *)" --prompt "Fix bug and commit changes"

# Build processes
--allowed-tools "Bash(npm run build),Read" --prompt "Build and check for errors"
```

## 🌐 Web & External Data

### WebSearch
**Purpose**: Search the web for current information  
**Security**: Low risk - information gathering  
**Syntax**: `WebSearch`

**Capabilities**:
- Up-to-date information beyond Claude's knowledge cutoff
- Domain filtering (include/exclude specific sites)
- Only available in the US
- Returns formatted search result blocks

**Usage Examples**:
```bash
# Technology research
--allowed-tools "WebSearch,Read" --prompt "Research latest React best practices and update our code"

# Documentation lookup
--allowed-tools "WebSearch" --prompt "Find official documentation for this API"
```

### WebFetch
**Purpose**: Fetch and analyze content from URLs  
**Security**: Low risk - content retrieval  
**Syntax**: `WebFetch`

**Capabilities**:
- Fetches URL content and converts HTML to markdown
- Self-cleaning 15-minute cache
- Handles redirects and informs about host changes
- Read-only operation

**Usage Examples**:
```bash
# Documentation analysis
--allowed-tools "WebFetch,Write" --prompt "Fetch API docs and create local reference"

# Content analysis
--allowed-tools "WebFetch" --prompt "Analyze this webpage for accessibility issues"
```

## 📓 Specialized Operations

### NotebookRead / NotebookEdit
**Purpose**: Jupyter notebook operations  
**Security**: NotebookRead (safe), NotebookEdit (high risk)  
**Syntax**: `NotebookRead`, `NotebookEdit`

**Capabilities**:
- Read/edit Jupyter notebook (.ipynb) files
- Cell-by-cell operations with cell IDs
- Support for code and markdown cells
- Insert, delete, and replace operations

### TodoWrite
**Purpose**: Task management and progress tracking  
**Security**: Low risk - task organization  
**Syntax**: `TodoWrite`

**Capabilities**:
- Structured task list management
- Task states: pending, in_progress, completed
- Priority levels: high, medium, low
- Progress tracking for complex workflows

### ExitPlanMode
**Purpose**: Transition from planning to implementation  
**Security**: Safe - workflow control  
**Syntax**: `ExitPlanMode`

**Usage**: Used when Claude needs to present implementation plans for user approval before proceeding with code changes.

## 🔒 Security Patterns & Best Practices

### Safe Analysis Mode
**Tools**: `Read`, `Grep`, `Glob`, `LS`, `WebSearch`, `WebFetch`  
**Use Case**: Code review, security analysis, documentation review  
**Risk Level**: ✅ **Low**

```bash
# Code security audit
--allowed-tools "Read,Grep,WebSearch" --prompt "Audit code for security vulnerabilities"

# Project analysis
--allowed-tools "Read,Glob,LS" --prompt "Analyze project structure and dependencies"
```

### Controlled Editing Mode
**Tools**: `Read`, `Edit`, `Write`  
**Use Case**: Bug fixes, small refactoring, documentation updates  
**Risk Level**: ⚠️ **Medium**

```bash
# Bug fix workflow
--allowed-tools "Read,Edit" --prompt "Fix the validation error in user registration"

# Documentation updates
--allowed-tools "Read,Write,Edit" --prompt "Update API documentation for new endpoints"
```

### Development Mode
**Tools**: `Read`, `Edit`, `MultiEdit`, `Write`, `Bash(git *)`  
**Use Case**: Feature development, comprehensive refactoring  
**Risk Level**: 🔸 **High**

```bash
# Feature development
--allowed-tools "Read,Edit,Write,Bash(git *)" --prompt "Implement user profile feature"

# Refactoring with git integration
--allowed-tools "Read,MultiEdit,Bash(git commit:*)" --prompt "Refactor auth system and commit changes"
```

### Full Development Mode
**Tools**: `Read`, `Edit`, `MultiEdit`, `Write`, `Bash(*)`  
**Use Case**: Complex system changes, build processes, testing  
**Risk Level**: 🔴 **Very High**

```bash
# Complete development workflow
--allowed-tools "Read,Edit,Write,Bash(*)" --prompt "Implement feature with full testing and deployment"

# System maintenance
--allowed-tools "Bash(*),Read,Edit" --prompt "Update dependencies and fix compatibility issues"
```

### Restricted System Access
**Tools**: `Bash(pattern)` with specific patterns  
**Use Case**: Controlled system operations  
**Risk Level**: 🔸 **High** (but controlled)

```bash
# Testing only
--allowed-tools "Read,Bash(npm test),Bash(npm run lint)" --prompt "Run all tests and linting"

# Git operations only
--allowed-tools "Read,Edit,Bash(git *)" --prompt "Make changes and handle git workflow"

# Specific commands only
--allowed-tools "Bash(docker ps),Bash(docker logs *)" --prompt "Check container status"
```

## 🎯 Common Usage Patterns

### Code Review Workflow
```bash
--allowed-tools "Read,Grep,WebSearch" 
--prompt "Review this pull request for security, performance, and best practices"
--max-turns 10
```

### Bug Fix Workflow
```bash
--allowed-tools "Read,Edit,Bash(npm test),Bash(git commit:*)"
--prompt "Fix the authentication bug and commit the solution"
--max-turns 20
```

### Documentation Generation
```bash
--allowed-tools "Read,Write,WebFetch"
--prompt "Generate comprehensive API documentation"
--max-turns 30
```

### Refactoring Workflow
```bash
--allowed-tools "Read,MultiEdit,Bash(npm test)"
--prompt "Refactor the user service to use modern async/await patterns"
--max-turns 50
```

### Security Audit
```bash
--allowed-tools "Read,Grep,WebSearch"
--prompt "Perform comprehensive security audit focusing on authentication and data validation"
--max-turns 20
```

## ⚠️ Tool Restrictions & Gotchas

### Permission Syntax Rules
- `ToolName` - permit every action with the tool
- `ToolName(*)` - permit any argument (same as above)
- `ToolName(pattern)` - permit only matching calls
- Deny rules sit on top of allow rules
- Claude walks the permission list from first to last

### Common Pitfalls
1. **Edit Tool**: Fails if old_string is not unique - use MultiEdit or provide more context
2. **Bash Tool**: Don't use `find`, `grep`, `cat` etc. - use specialized tools instead
3. **File Paths**: Always use absolute paths with LS tool
4. **MultiEdit**: All edits must be valid or none are applied (atomic operation)
5. **Timeout Handling**: Commands timeout after 2 minutes by default (configurable up to 10 minutes)

### Best Practices
1. **Start Restrictive**: Begin with minimal tools and add as needed
2. **Use Patterns**: Leverage tool patterns like `Bash(git *)` for controlled access
3. **Set Limits**: Always use `--max-turns` to prevent runaway operations
4. **Review Changes**: Never skip reviewing code changes before committing
5. **Version Control**: Always work in version-controlled environments
6. **Test Integration**: Include testing tools in your allowed tools list

This comprehensive tool reference enables precise control over Claude Code's capabilities while maintaining security and enabling powerful automation workflows.