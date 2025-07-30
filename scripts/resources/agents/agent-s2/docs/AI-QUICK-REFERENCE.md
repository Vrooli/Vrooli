# Agent-S2 AI Quick Reference

## 🚀 Quick Start

### 1. Check AI Status
```bash
curl http://localhost:4113/ai/status
```

### 2. Run Diagnostics (if issues)
```bash
curl http://localhost:4113/ai/diagnostics | jq .
```

### 3. Execute AI Actions
```bash
# Simple task
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{"task": "Take a screenshot and save it"}'

# With specific actions
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{"task": "Click at position 500,300 then type Hello World"}'
```

## 🔧 Troubleshooting

### AI Not Available?

1. **Check diagnostics first**:
   ```bash
   curl http://localhost:4113/ai/diagnostics | jq '.recommendations'
   ```

2. **Common fixes**:
   - Install Ollama: `curl -fsSL https://ollama.com/install.sh | sh`
   - Start Ollama: `ollama serve`
   - Pull vision model: `ollama pull llama3.2-vision:11b`
   - Enable AI: `export AGENTS2_ENABLE_AI=true`

3. **Container issues**:
   ```bash
   # Fix permissions
   docker exec -u root agent-s2 bash -c "mkdir -p /data/sessions/profiles && chown -R agents2:agents2 /data"
   docker restart agent-s2
   ```

## 📊 Understanding Responses

### Successful AI Action
```json
{
  "success": true,
  "task": "Your task description",
  "summary": "AI analyzed and executed task: ...",
  "actions_taken": [
    {
      "action": "click",
      "parameters": {"x": 500, "y": 300},
      "result": "Clicked at position",
      "status": "success"
    }
  ],
  "reasoning": "AI's explanation"
}
```

### Error Response (Enhanced)
```json
{
  "detail": {
    "error": "AI service not available",
    "reason": "Specific reason for failure",
    "category": "connection_failed|model_not_found|disabled",
    "suggestions": ["Step 1 to fix", "Step 2 to fix"],
    "alternatives": {
      "endpoints": {
        "screenshot": "/screenshot",
        "mouse": "/mouse/click",
        "keyboard": "/keyboard/type"
      }
    },
    "diagnostics": "Use GET /ai/diagnostics"
  }
}
```

## 🎯 Best Practices

1. **Always use vision models** for screen interaction tasks
2. **Check `/ai/diagnostics`** when troubleshooting
3. **Let auto-detection work** - don't override unless necessary
4. **Use alternative endpoints** when AI is unavailable

## 🔗 API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/ai/status` | GET | Check AI availability |
| `/ai/diagnostics` | GET | Detailed troubleshooting |
| `/ai/action` | POST | Execute AI-driven actions |
| `/ai/command` | POST | Natural language commands |
| `/ai/plan` | POST | Generate action plans |
| `/ai/analyze` | POST | Analyze screen content |

## 🐛 Common Issues

### "Permission denied: '/data'"
```bash
docker exec -u root agent-s2 bash -c "mkdir -p /data && chown agents2:agents2 /data"
docker restart agent-s2
```

### "Cannot connect to Ollama"
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# If not, start it
ollama serve
```

### "Model not found"
```bash
# List available models
ollama list

# Pull recommended model
ollama pull llama3.2-vision:11b
```

## 📝 Example Workflows

### Screenshot and Save
```bash
curl -X POST http://localhost:4113/ai/action \
  -d '{"task": "Take a screenshot and save it as desktop.png"}'
```

### Multi-Step Automation
```bash
curl -X POST http://localhost:4113/ai/action \
  -d '{
    "task": "Open a text editor, type Hello World, then save the file"
  }'
```

### Targeted Actions
```bash
curl -X POST http://localhost:4113/ai/command \
  -d '{
    "command": "Click on the Firefox icon",
    "target_app": "firefox"
  }'
```