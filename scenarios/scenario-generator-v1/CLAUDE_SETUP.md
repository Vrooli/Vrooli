# Claude Code Integration Setup Guide

## ✅ Integration Status
The scenario-generator-v1 now has **fully functional Claude Code integration** using the `resource-claude-code` command.

## 🔧 What Was Fixed
1. **Command Path**: Changed from incorrect `vrooli resource claude chat` to correct `resource-claude-code run`
2. **Argument Format**: Fixed to pass prompts as command-line arguments instead of stdin
3. **Error Handling**: Added proper error detection for rate limits, auth issues, and command availability
4. **String Literals**: Fixed Go syntax issues with backtick literals in prompt templates

## 🚀 Setup Instructions

### 1. Ensure Claude Code Resource is Installed
```bash
# Check if resource-claude-code is available
resource-claude-code status

# If not installed, install it:
vrooli resource install claude-code
```

### 2. Authenticate Claude Code
```bash
# Login to Claude (required for generation to work)
claude auth login

# Verify authentication
resource-claude-code status
```

### 3. Set Resource Tier (Optional)
```bash
# Set your subscription tier for accurate rate limits
resource-claude-code set-tier pro  # or 'free', 'team', etc.
```

### 4. Test the Integration
```bash
cd /home/matthalloran8/Vrooli/scenarios/scenario-generator-v1/api
go run test-claude.go
```

## 📝 Configuration Notes

### Environment Variables
The ClaudeClient automatically uses these environment variables:
- `VROOLI_ROOT`: Root directory for Vrooli (defaults to ~/Vrooli)
- `HOME`: Used as fallback if VROOLI_ROOT not set

### Timeout Configuration
- Default timeout: 10 minutes per Claude request
- Retry attempts: 3 with exponential backoff
- Kill timeout: 30 seconds after SIGTERM

### Rate Limiting
The integration handles rate limits automatically:
- Detects rate limit errors from Claude
- Uses exponential backoff between retries
- Returns clear error messages when limits are hit

## 🔍 Troubleshooting

### Common Issues and Solutions

1. **"claude not authenticated" error**
   - Run: `claude auth login`
   - Follow the authentication prompts

2. **"resource-claude-code not found" error**
   - Install the resource: `vrooli resource install claude-code`
   - Ensure ~/.local/bin is in your PATH

3. **Rate limit errors**
   - Wait for the cooldown period
   - Check your tier limits: `resource-claude-code usage`
   - Consider upgrading tier if needed

4. **Empty responses**
   - Check Claude Code logs: `resource-claude-code logs`
   - Verify the prompt is being passed correctly
   - Ensure Claude service is running: `resource-claude-code status`

## 💡 Usage in Pipeline

The pipeline uses Claude Code in three phases:

1. **Planning Phase** (`planning.go`)
   - Generates architectural plans and identifies resources
   - Uses iterative refinement (1-5 iterations)

2. **Implementation Phase** (`implementation.go`)
   - Generates complete scenario files
   - Parses code blocks from responses
   - Supports bug fixing iterations

3. **Validation Phase** (`validation.go`)
   - Analyzes validation errors
   - Generates fixes for failed validations
   - Iterates until validation passes

## 🧪 Testing Claude Integration

Run the provided test to verify everything works:
```bash
cd api
./test-claude
```

Expected output:
```
Testing Claude Code integration...
✅ Claude integration test completed successfully!
```

## 🎯 Production Readiness

The integration is now production-ready with:
- ✅ Proper command execution
- ✅ Error handling and retries
- ✅ Rate limit detection
- ✅ Timeout management
- ✅ Code block parsing
- ✅ Streaming simulation

The scenario-generator-v1 can now autonomously generate complete Vrooli scenarios using Claude Code!