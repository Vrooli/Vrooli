# LiteLLM Routing Decision Flow

```
📥 Request Comes In
         │
         ▼
🤔 Check Model Parameter
         │
    ┌────┴────┐
    ▼         ▼
"auto"    Specific Model
 │           │
 │           ▼
 │      🎯 Route directly to
 │         specified model
 │           │
 │           ▼
 │      ✅ Return response
 │
 ▼
🧮 Cost-Based Routing
         │
         ▼
💰 Check Model Costs:
   • Ollama: $0.00
   • OpenAI: $0.002/1K
   • Anthropic: $0.003/1K
   • OpenRouter: varies
         │
         ▼
🔄 Try Models in Order:
         │
    ┌────┴────┐
    ▼         ▼
🦙 Try Ollama  ❌ Ollama Failed?
    │              │
    ▼              ▼
✅ Success?    🔄 Try Next: OpenAI
    │              │
    ▼              ▼
🎉 Return      ✅ Success?
               │
               ▼
           🔄 Try Next: Anthropic
               │
               ▼
           ✅ Success?
               │
               ▼
           🔄 Try Next: OpenRouter
               │
               ▼
           😞 All Failed
```

## Decision Factors

### 1. **API Key Availability**
```
✅ Has OpenAI key → Include in routing
❌ No Anthropic key → Skip Anthropic
✅ Has OpenRouter key → Use as fallback
```

### 2. **Model Health Status**
```
🟢 Ollama running → Try first
🔴 OpenAI rate limited → Skip temporarily
🟡 Anthropic slow → Lower priority
```

### 3. **Cost Optimization**
```
FREE: Ollama models (always preferred)
$: Cheap API models (gpt-3.5-turbo)
$$: Expensive models (gpt-4, claude-opus)
```

### 4. **Fallback Configuration**
```yaml
# Custom fallback chains
fallbacks:
  - llama2-local: ["gpt-3.5-turbo", "claude-3-haiku"]
  - gpt-4: ["claude-3-sonnet", "gpt-3.5-turbo"]
```

## Example Routing Decisions

### Simple Task: "Hello"
```
1. 🦙 llama2-local (FREE) → ✅ Success
   Cost: $0.00
```

### Complex Task: "Write a web app"
```
1. 🦙 llama2-local (FREE) → ❌ Limited capability
2. 🤖 gpt-3.5-turbo ($0.002) → ✅ Success
   Cost: ~$0.01
```

### When Ollama is Down
```
1. 🦙 llama2-local → ❌ Service unavailable
2. 🤖 gpt-3.5-turbo → ✅ Success
   Automatic failover worked!
```

### No API Keys Configured
```
1. 🦙 llama2-local (FREE) → ✅ Only option available
   Falls back to local-only mode
```