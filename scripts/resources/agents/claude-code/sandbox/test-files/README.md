# Claude Code Sandbox Test Files

This directory contains example files for testing Claude Code in the sandbox environment.

## 📁 What Goes Here

Place any files you want to test with Claude Code in this directory:
- Source code for analysis
- Configuration files
- Documentation to review
- Scripts to debug
- Any text-based files

## 🔒 Security Note

**This directory is the ONLY location accessible to Claude Code in sandbox mode.**

Benefits:
- Your main codebase remains untouched
- Test risky operations safely
- Experiment with `--dangerously-skip-permissions`
- Isolate untrusted code

## 📋 Example Files Included

### `example.js`
JavaScript file with various code patterns including:
- Classes and methods
- Security vulnerabilities (eval usage)
- Missing error handling
- Async operations

**Test commands:**
```bash
# Analyze for issues
./manage.sh --action sandbox --prompt "Review example.js for security issues"

# Fix bugs
./manage.sh --action sandbox --prompt "Fix the division by zero bug in example.js"

# Add tests
./manage.sh --action sandbox --prompt "Write unit tests for Calculator class"
```

### `test-patterns.py`
Python file demonstrating:
- Type hints
- Performance issues (O(n²) loops)
- Security vulnerabilities (command injection)
- Missing error handling

**Test commands:**
```bash
# Security audit
./manage.sh --action sandbox --prompt "Perform security audit on test-patterns.py"

# Refactor for performance
./manage.sh --action sandbox --prompt "Optimize the process_records method"

# Add error handling
./manage.sh --action sandbox --prompt "Add proper error handling throughout the file"
```

## 🎯 Testing Scenarios

### 1. Code Review
```bash
./manage.sh --action sandbox --prompt "Review all files in the workspace for best practices"
```

### 2. Bug Fixing
```bash
./manage.sh --action sandbox --prompt "Find and fix all bugs in the JavaScript files"
```

### 3. Security Analysis
```bash
./manage.sh --action sandbox --prompt "Identify security vulnerabilities and suggest fixes"
```

### 4. Documentation Generation
```bash
./manage.sh --action sandbox --prompt "Generate comprehensive documentation for all code"
```

### 5. Refactoring
```bash
./manage.sh --action sandbox --prompt "Refactor code to follow SOLID principles"
```

## 💡 Tips

1. **Start Small**: Test with simple files first
2. **Copy, Don't Move**: Keep originals safe, copy files here for testing
3. **Clean Regularly**: Remove old test files to avoid confusion
4. **Version Control**: This directory is gitignored - files here are temporary

## ⚠️ Important

- Files here are accessible to Claude Code with `--dangerously-skip-permissions`
- Don't put sensitive data (passwords, keys, personal info) in test files
- Remember this is for testing - validate all changes before production use

## 🧪 Advanced Testing

### Multi-file Projects
Create subdirectories to test project structures:
```bash
mkdir -p test-files/my-project/{src,tests,docs}
cp -r ~/projects/example/* test-files/my-project/
```

### Integration Testing
Test Claude Code with different file types:
- `.json` - Configuration files
- `.md` - Documentation
- `.yaml` - CI/CD configs
- `.sql` - Database schemas
- `.sh` - Shell scripts

### Batch Operations
```bash
# Process multiple files
./manage.sh --action sandbox --sandbox-command batch \
  --prompt "Add error handling to all Python files" \
  --max-turns 100
```

Remember: The sandbox is your safe space to experiment. Use it to understand Claude Code's capabilities before running on real projects!