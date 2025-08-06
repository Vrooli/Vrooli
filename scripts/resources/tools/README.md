# Resource Interface Compliance Tool

## Overview

The `fix-interface-compliance.sh` tool automatically detects and fixes common interface compliance issues in Vrooli resource `manage.sh` scripts. It ensures all resources follow consistent patterns and implement required functions according to the interface contracts.

## 🎯 What It Fixes

### **Automatic Fixes**
- ✅ **Missing Functions**: Generates stubs for required actions (install, start, stop, status, logs)
- ✅ **Permission Issues**: Makes scripts executable (`chmod +x`)
- ✅ **Action Patterns**: Standardizes quoted/unquoted action patterns in case statements
- ✅ **Missing Logs Functions**: Adds missing `logs` functions when referenced in case statements

### **Validation Checks**
- ✅ **Contract Compliance**: Validates against category-specific interface contracts
- ✅ **Function Existence**: Checks that case statement actions have corresponding functions
- ✅ **File Structure**: Verifies required directories and files exist
- ✅ **Error Handling**: Validates proper bash error handling patterns

## 🚀 Quick Start

### **Basic Usage**
```bash
# Dry-run to see what would be fixed
./fix-interface-compliance.sh --resource whisper --dry-run

# Apply fixes
./fix-interface-compliance.sh --resource whisper --apply

# Comprehensive fixing with all features
./fix-interface-compliance.sh --resource n8n --apply --verbose
```

### **Integrated Validation & Fixing**
```bash
# Validate and auto-fix in one command
./validate-interfaces.sh --resource whisper --fix
```

## 📚 Command Reference

### **Core Options**
| Option | Description | Default |
|--------|-------------|---------|
| `--resource <name>` | Fix specific resource only | Required |
| `--dry-run` | Show what would be changed (safe) | `true` |
| `--apply` | Actually apply the fixes | `false` |
| `--verbose` | Detailed output | `false` |

### **Fix Control Options**
| Option | Description | Default |
|--------|-------------|---------|
| `--fix-permissions` | Fix script executable permissions | `true` |
| `--no-permissions` | Skip permission fixing | `false` |
| `--fix-patterns` | Fix quoted action patterns | `true` |
| `--no-patterns` | Skip pattern fixing | `false` |

### **Advanced Options**
| Option | Description | Default |
|--------|-------------|---------|
| `--actions <list>` | Specific actions to generate | From contracts |
| `--category <cat>` | Override resource category | Auto-detect |
| `--backup` | Create backup files | `true` |
| `--no-backup` | Skip creating backups | `false` |
| `--force` | Apply even with warnings | `false` |

## 🔧 Examples

### **Basic Compliance Fixing**
```bash
# Check what needs fixing (safe)
./fix-interface-compliance.sh --resource whisper --dry-run

# Output:
# DRY RUN - Showing all fixes that would be applied:
# ========== PERMISSION FIX ==========
# PERMISSION FIX: Make script executable (chmod +x)
#
# ========== PATTERN FIXES ==========  
# PATTERN FIX: Remove quotes from 12 action patterns
#   Example: "install") -> install)
#
# ========== MISSING FUNCTION STUBS ==========
# Function stub for: logs
# ...
```

### **Apply Fixes with Backup**
```bash
./fix-interface-compliance.sh --resource n8n --apply

# Output:
# [INFO] Applying 3 fix(es) to n8n...
# [SUCCESS] ✅ Backup created: .../n8n/manage.sh.backup.20250805_213045
# [SUCCESS] ✅ Fixed permissions for: .../n8n/manage.sh
# [SUCCESS] ✅ Fixed action patterns in: .../n8n/manage.sh  
# [SUCCESS] ✅ Applied function stub fixes to .../n8n/manage.sh
# [SUCCESS] ✅ Successfully applied all 3 fix(es) to n8n
```

### **Selective Fixing**
```bash
# Only fix permissions, skip patterns
./fix-interface-compliance.sh --resource vault --apply --no-patterns

# Only fix specific actions
./fix-interface-compliance.sh --resource comfyui --actions install,start,stop --apply

# Fix without backup (not recommended)
./fix-interface-compliance.sh --resource whisper --apply --no-backup
```

## 📋 Interface Contracts

The tool validates against versioned interface contracts located in `scripts/resources/contracts/v1.0/`:

### **Core Contract** (`core.yaml`)
Required for **ALL** resources:
- `install()` - Install the resource and dependencies
- `start()` - Start the service  
- `stop()` - Stop the service
- `status()` - Check service status and health
- `logs()` - Display service logs

### **Category-Specific Contracts**
- **AI Resources** (`ai.yaml`) - ollama, whisper, unstructured-io, comfyui
- **Automation** (`automation.yaml`) - n8n, node-red, windmill, huginn  
- **Storage** (`storage.yaml`) - postgres, redis, vault, qdrant, minio
- **Agents** (`agents.yaml`) - browserless, claude-code, agent-s2
- **Search** (`search.yaml`) - searxng
- **Execution** (`execution.yaml`) - judge0

## 🔍 Generated Code Examples

### **Function Stubs**
```bash
#######################################
# Install the whisper resource
# Arguments:
#   None  
# Returns:
#   0 - Success
#   1 - Error
#   2 - Already in desired state (skip)
#######################################
whisper::install() {
    log::info "Installing whisper..."
    
    # TODO: Add installation logic here
    # Example patterns:
    # - Check if already installed: if resource_installed; then return 2; fi
    # - Install dependencies: install_dependencies
    # - Download/setup service: setup_service
    # - Configure service: configure_service
    # - Start service: whisper::start
    
    log::success "whisper installation completed"
}
```

### **Pattern Fixes**
```bash
# Before (quoted patterns):
case "$ACTION" in
    "install")
        whisper::install
        ;;
    "start")
        whisper::start_container
        ;;

# After (unquoted patterns):
case "$ACTION" in
    install)
        whisper::install
        ;;
    start)
        whisper::start_container
        ;;
```

## 🛡️ Safety Features

### **Dry-Run by Default**
- All operations default to dry-run mode
- Shows exactly what would be changed
- Must explicitly use `--apply` to make changes

### **Automatic Backups**
- Creates timestamped backups before applying changes
- Backup format: `manage.sh.backup.YYYYMMDD_HHMMSS`
- Can be disabled with `--no-backup` if needed

### **Syntax Validation**
- All generated code is validated with `bash -n`
- Changes are reverted if syntax errors are detected
- Prevents breaking existing scripts

### **Incremental Fixes**
- Each fix type can be controlled independently
- Failed fixes don't prevent other fixes from applying
- Detailed reporting of what succeeded/failed

## 🔄 Integration with Validation

### **Main Validation Command**
The tool integrates with the main validation system:

```bash
# Validate and auto-fix failures
./validate-interfaces.sh --fix

# Target specific resources
./validate-interfaces.sh --resource whisper --fix
```

### **CI/CD Integration**
```yaml
# Example GitHub Actions integration
- name: Validate Resource Interfaces
  run: |
    cd scripts/resources
    ./validate-interfaces.sh --level standard --format junit
    
- name: Auto-fix Interface Issues  
  run: |
    cd scripts/resources
    ./validate-interfaces.sh --fix --apply
    
- name: Commit Fixes
  run: |
    git add -A
    git commit -m "🔧 Auto-fix resource interface compliance" || exit 0
```

## 🐛 Troubleshooting

### **Common Issues**

**Tool says "No resource found"**
```bash
# Check resource name and location
find scripts/resources -name "*whisper*" -type d
./fix-interface-compliance.sh --resource whisper --verbose
```

**Generated functions have TODOs**  
```bash
# This is expected - implement the actual logic
# Use existing functions as reference:
grep -A 20 "whisper::start_container" scripts/resources/ai/whisper/manage.sh
```

**Backup files accumulating**
```bash
# Clean old backups (>7 days)
find scripts/resources -name "manage.sh.backup.*" -mtime +7 -delete
```

**Permission denied errors**
```bash
# Run with proper permissions
chmod +x scripts/resources/tools/fix-interface-compliance.sh
./fix-interface-compliance.sh --resource whisper --apply
```

### **Verbose Debugging**
```bash
# Get detailed information about what the tool is doing
./fix-interface-compliance.sh --resource whisper --verbose --dry-run
```

## 🧪 Testing

### **Test the Tool**
```bash
# Test on a known working resource
./fix-interface-compliance.sh --resource ollama --dry-run

# Should output: "No compliance issues found"

# Test on a resource with issues  
./fix-interface-compliance.sh --resource whisper --dry-run

# Should show specific fixes needed
```

### **Validate Fixes**
```bash
# Apply fixes
./fix-interface-compliance.sh --resource whisper --apply

# Verify they worked
./validate-interfaces.sh --resource whisper

# Should show: "✅ whisper: Interface validation passed"
```

## 📈 Best Practices

### **Development Workflow**
1. **Check before coding**: Run dry-run to understand issues
2. **Apply incrementally**: Fix one type of issue at a time if needed
3. **Validate after**: Always run validation after applying fixes
4. **Keep backups**: Don't use `--no-backup` unless necessary

### **Team Usage**
1. **Document fixes**: Include compliance fixes in PR descriptions
2. **Review generated code**: Don't just accept TODOs - implement properly
3. **Test thoroughly**: Ensure fixes don't break existing functionality
4. **Share knowledge**: Update team on new compliance requirements

### **Maintenance**
1. **Update contracts**: Keep interface contracts up to date
2. **Monitor usage**: Track which resources need frequent fixes
3. **Improve patterns**: Enhance the tool based on common issues
4. **Clean backups**: Regularly clean old backup files

## 🚀 Future Enhancements

### **Recently Implemented**
- ✅ **Batch processing**: Fix multiple resources at once (implemented)
  - `--batch --resources "whisper,n8n,comfyui"`
  - `--batch --category automation`
  - `--batch --all`

### **Planned Features**
- **Smart delegation**: Auto-wire standard functions to existing implementations  
- **Configuration validation**: Check config file compliance
- **Help text generation**: Auto-generate proper help/usage text
- **Parallel processing**: Enhance batch processing with parallel execution for large codebases

### **Integration Improvements**
- **IDE integration**: VS Code extension for real-time compliance checking
- **Git hooks**: Pre-commit hooks for automatic compliance checking
- **Monitoring**: Track compliance metrics over time
- **Automated PRs**: Create pull requests for compliance fixes

---

## 📞 Support

- **Issues**: Report bugs in the main project repository
- **Questions**: Ask in team chat or create discussion topics
- **Contributions**: Follow standard PR process for improvements
- **Documentation**: Keep this README updated with new features

**Remember**: This tool is designed to help maintain consistency and quality across all Vrooli resources. Use it proactively to prevent issues rather than just reactively fixing them!