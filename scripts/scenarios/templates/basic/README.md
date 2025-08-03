# Basic Integration Test

> **Simple resource integration validation template**

## 🎯 Purpose

This template is designed for basic resource integration testing. Use it when you need to:
- Validate resource connectivity
- Test basic functionality
- Create internal development scenarios
- Learn the scenario system

## 📋 Usage

1. **Copy Template**:
   ```bash
   cp -r templates/basic/ my-test-scenario/
   cd my-test-scenario/
   ```

2. **Update Configuration**:
   - Edit `metadata.yaml` with your resources and requirements
   - Update `README.md` with scenario details
   - Modify `test.sh` with your integration tests

3. **Run Tests**:
   ```bash
   ./test.sh
   ```

## 🔧 Customization

### Required Changes
- **metadata.yaml**: Update scenario ID, resources, and testing requirements
- **test.sh**: Replace placeholder tests with actual validation logic
- **README.md**: Document your specific use case

### Example Resource Combinations
- **AI Testing**: `ollama` + basic prompt validation
- **Database Testing**: `postgres` + connection and query validation  
- **Workflow Testing**: `n8n` + webhook and workflow execution
- **Storage Testing**: `minio` + file upload and retrieval

## 🧪 Test Structure

The basic test template includes:
- ✅ Resource health validation
- ✅ Basic functionality testing
- ✅ Error handling validation
- ✅ Simple performance checks

## 📊 Success Criteria

A basic scenario passes when:
- All required resources are accessible
- Basic functionality works as expected
- Integration tests complete successfully
- No critical errors occur

---

*This template provides a foundation for resource integration testing. For business applications, use the [business template](../business/) or [AI-generation template](../ai-generation/).*