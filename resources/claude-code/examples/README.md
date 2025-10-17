# Claude Code Examples

This directory contains practical examples of using Claude Code for various development tasks.

## 📚 Example Categories

### 01. Basic Usage
- Interactive sessions
- Simple prompts and responses
- Command-line integration

### 02. Code Analysis
- Code review automation
- Security analysis
- Performance optimization

### 03. Development Workflows
- Test generation
- Documentation creation
- Refactoring assistance

### 04. Integration Patterns
- CI/CD pipeline integration
- Version control workflows
- Multi-resource automation

## 🚀 Getting Started

Before running examples, ensure Claude Code is installed:

```bash
# Check installation
./manage.sh --action status

# If not installed
./manage.sh --action install

# Verify functionality
claude --version
claude doctor
```

## ⚠️ Safety Notes

- **Review all changes** before committing to version control
- **Use version control** - commit working code before running examples
- **Start in test environments** for dangerous operations
- **Understand permissions** - avoid `--dangerously-skip-permissions` unless absolutely necessary

## 🔧 Example Usage Patterns

### Quick Single Commands
```bash
# Code explanation
echo "console.log('Hello World');" | claude -p "Explain this code"

# Quick analysis
find . -name "*.js" | claude -p "Find potential security issues"
```

### Management Script Usage
```bash
# Simple task
./manage.sh --action run --prompt "Add error handling to this function"

# Complex refactoring
./manage.sh --action batch --prompt "Modernize all React components" --max-turns 50
```

### Session Management
```bash
# Resume previous work
./manage.sh --action session

# Continue specific session
./manage.sh --action session --session-id abc123def456
```

## 🛡️ Security Best Practices

### Safe Tool Usage
```bash
# Read-only analysis
./manage.sh --action run --prompt "Analyze code" --allowed-tools "Read,Grep"

# Limited editing
./manage.sh --action run --prompt "Fix typos" --allowed-tools "Edit" --max-turns 5
```

### Dangerous Operations (Use with extreme caution)
```bash
# Only in isolated test environments
./manage.sh --action run --prompt "System task" \
  --dangerously-skip-permissions yes \
  --allowed-tools "Edit,Read" \
  --max-turns 10
```

## 📁 Example Files

Each example includes:
- **README.md** - Explanation and instructions
- **example.sh** - Runnable script
- **sample-input** - Test files when applicable
- **expected-output** - What to expect

Navigate to specific example directories for detailed instructions and runnable code.

## 🔄 Contributing Examples

When adding new examples:
1. Create descriptive directory names
2. Include comprehensive README
3. Test examples thoroughly
4. Document any prerequisites
5. Include safety warnings for dangerous operations