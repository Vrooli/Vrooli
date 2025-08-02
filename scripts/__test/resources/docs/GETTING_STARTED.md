# 🚀 Getting Started with Vrooli Resource Testing

**Quick Start Guide for Developers - Get up and running in 2 minutes!**

## 🎯 30-Second Quick Test

Test if everything works with one command:

```bash
# Navigate to the test directory
cd /home/matthalloran8/Vrooli/scripts/resources/tests

# Run a quick test on Ollama (if available)
./run.sh --resource ollama

# Alternative: Test whatever resources are running
./run.sh --single-only
```

**Expected Output**: ✅ Test results showing which resources are healthy and working.

---

## 🏗️ What Am I Looking At?

The Vrooli Resource Test Framework tests your AI/automation services **without mocking** - it talks to real running services to make sure everything actually works together.

### **Simple Mental Model**:
```
Your Services (Ollama, n8n, etc.) → Test Framework → "Yes, it all works!" ✅
```

### **Directory Structure (The Important Parts)**:
```
tests/
├── run.sh                    # ← START HERE: Main test runner
├── GETTING_STARTED.md        # ← You are here
├── single/                   # Tests for individual services
│   ├── ai/ollama.test.sh     # Test just Ollama
│   ├── automation/n8n.test.sh # Test just n8n
│   └── ...
├── scenarios/                # Test multiple services working together
│   ├── multi-modal-ai-assistant/  # Complete business workflows
│   └── ...
└── framework/                # The engine (you usually don't need to touch this)
```

---

## 🎮 5-Minute Developer Tour

### **Step 1: See What's Available**
```bash
# See what resources are running
./run.sh --list-scenarios

# See what individual resources can be tested
ls single/*/
```

### **Step 2: Test One Thing**
```bash
# Test just one service (replace 'ollama' with any available resource)
./run.sh --resource ollama

# More detailed output
./run.sh --resource ollama --verbose
```

### **Step 3: Test Everything**
```bash
# Test all available resources
./run.sh --single-only

# Test business scenarios (multi-service workflows)
./run.sh --scenarios-only
```

### **Step 4: Test Specific Workflows**
```bash
# Test AI-focused scenarios
./run.sh --scenarios "category=ai-assistance"

# Test specific complexity level
./run.sh --scenarios "complexity=intermediate"
```

---

## 📋 Copy-Paste Examples for Common Tasks

### **"I want to test if my setup works"**
```bash
./run.sh --single-only --timeout 300 --verbose
```

### **"I want to test just AI services"**
```bash
./run.sh --resource ollama --resource whisper --resource comfyui
```

### **"I want to see a complete business workflow"**
```bash
./run.sh --scenarios "multi-modal-ai-assistant"
```

### **"I want detailed debugging info"**
```bash
./run.sh --debug --resource ollama
```

### **"I want JSON output for CI/CD"**
```bash
./run.sh --output-format json --single-only > test-results.json
```

### **"I want to stop on first failure"**
```bash
./run.sh --fail-fast --single-only
```

---

## 🛠️ "How Do I..." Quick Reference

### **Add a Test for My New Service?**
1. Look at existing test: `single/ai/ollama.test.sh`
2. Copy the pattern for your service category
3. Replace service-specific details (URL, endpoints, etc.)
4. Test with: `./run.sh --resource your-service`

### **Debug Why a Test Fails?**
```bash
# Run with debug output
./run.sh --resource your-service --debug --verbose

# Check if the service is actually running
curl http://localhost:YOUR_PORT/health

# Check the discovery output
../index.sh --action discover
```

### **Run Tests in CI/CD?**
```bash
# Headless testing with JSON output
./run.sh --output-format json --single-only --fail-fast --timeout 600 > results.json
```

### **Test Multiple Services Working Together?**
```bash
# See available integration scenarios
./run.sh --list-scenarios

# Run specific integration test
./run.sh --scenarios "multi-resource-pipeline"
```

---

## 🔍 Understanding Test Results

### **✅ Success Output**:
```
✅ ollama tests passed
✅ Single-resource tests complete: 3/3 passed
```

### **❌ Failure Output**:
```
❌ ollama tests failed
💡 Suggestions:
   • Check if Ollama is running on port 11434
   • Verify models are installed: curl http://localhost:11434/api/tags
```

### **⏭️ Skipped Output**:
```
⏭️ whisper tests skipped - required resources unavailable
💡 Suggestion: Check resource health with './scripts/resources/index.sh --action discover'
```

---

## 🚨 Common Issues & Quick Fixes

### **"No tests run" or "All tests skipped"**
**Problem**: Services aren't running or discoverable
**Solution**:
```bash
# Check what's actually running
../index.sh --action discover

# Start missing services
../index.sh --action install --resources "ollama,n8n"
```

### **"Test timeout" errors**
**Problem**: Tests are taking too long
**Solution**:
```bash
# Increase timeout (default is 300s)
./run.sh --timeout 900 --resource slow-service
```

### **"Permission denied" or file not found**
**Problem**: Scripts aren't executable or paths are wrong
**Solution**:
```bash
# Make scripts executable
chmod +x run.sh
chmod +x single/**/*.test.sh

# Run from the correct directory
cd /home/matthalloran8/Vrooli/scripts/resources/tests
```

---

## 🎯 What's Next?

### **For Basic Testing**: You're all set! Use the copy-paste examples above.

### **For Advanced Usage**: Check out:
- [COMMON_PATTERNS.md](COMMON_PATTERNS.md) - More detailed examples
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Comprehensive problem solving
- [framework/README.md](framework/README.md) - Deep technical details

### **For Contributing**: 
- Look at `single/ai/ollama.test.sh` as the gold standard
- Follow the same patterns for new resources
- Business scenarios in `scenarios/` show complete workflows

---

## 💡 Pro Tips

1. **Start Small**: Always test one resource first before testing everything
2. **Use Verbose Mode**: `--verbose` shows you what's actually happening  
3. **Check Discovery First**: If tests fail, run `../index.sh --action discover` to see what's available
4. **Read the Output**: The framework gives helpful suggestions when things fail
5. **JSON for Automation**: Use `--output-format json` for CI/CD integration

---

**🎉 You're Ready!** The framework is designed to be helpful - it will guide you through problems with suggestions and clear error messages. Happy testing!