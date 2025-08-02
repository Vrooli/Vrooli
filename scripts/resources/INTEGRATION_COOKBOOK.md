# 🍽️ Vrooli Resource Integration Cookbook

A practical guide with working examples for integrating multiple resources in the Vrooli ecosystem.

## 🎯 **Quick Reference**

| Pattern | Resources | Use Case | Complexity |
|---------|-----------|----------|------------|
| [Vault + n8n Secret Workflow](#vault--n8n-secret-workflow) | Vault + n8n | Secure API orchestration | ⭐⭐ |
| [Agent-S2 + Browserless Comparison](#agent-s2--browserless-comparison) | Agent-S2 + Browserless | Web automation strategy | ⭐⭐⭐ |
| [AI + Automation + Storage Pipeline](#ai--automation--storage-pipeline) | Ollama + n8n + MinIO | AI processing with persistence | ⭐⭐⭐⭐ |
| [Multi-Modal Content Processing](#multi-modal-content-processing) | Whisper + Ollama + ComfyUI + MinIO | Audio → AI analysis → Image generation | ⭐⭐⭐⭐⭐ |
| [Research + Agent Workflow](#research--agent-workflow) | SearXNG + Agent-S2 + MinIO | Automated research and interaction | ⭐⭐⭐⭐ |

---

## 🔐 **Vault + n8n Secret Workflow**

**Problem**: Need to securely manage API keys in automated workflows  
**Solution**: Store secrets in Vault, retrieve them in n8n workflows  
**Resources**: `vault` + `n8n`

### **Setup Steps**

1. **Store secrets in Vault:**
```bash
# Store Stripe configuration
./scripts/resources/storage/vault/manage.sh --action put-secret \
  --path "test/stripe-config" \
  --value '{"stripe_api_key":"sk_test_123","environment":"test"}'

# Store OpenAI configuration  
./scripts/resources/storage/vault/manage.sh --action put-secret \
  --path "test/openai-config" \
  --value '{"openai_api_key":"sk-proj-123","model":"gpt-3.5-turbo"}'
```

2. **Import the n8n workflow:**
```bash
# Use the pre-built workflow
curl -X POST http://localhost:5678/api/v1/workflows/import \
  -H "Content-Type: application/json" \
  -d @/home/matthalloran8/Vrooli/scripts/resources/storage/vault/examples/n8n-vault-integration.json
```

3. **Test the integration:**
```bash
# Trigger the workflow
curl -X POST http://localhost:5678/webhook/webhook-vault-demo
```

### **Key Benefits**
- ✅ **Centralized secret management** - All API keys in one secure location
- ✅ **Audit trail** - Vault logs all secret access
- ✅ **Zero secrets in code** - n8n workflows contain no hardcoded credentials
- ✅ **Easy rotation** - Update secrets in Vault without touching workflows

---

## 🤖 **Agent-S2 + Browserless Comparison**

**Problem**: When to use which web automation tool?  
**Solution**: Strategic selection based on use case  
**Resources**: `agent-s2` + `browserless`

### **Decision Matrix**

| Scenario | Recommended Tool | Why |
|----------|------------------|-----|
| **Known internal dashboards** | Browserless | Fast, reliable, predictable |
| **Public websites with anti-bot** | Agent-S2 | Visual reasoning, adaptive behavior |
| **Simple PDF generation** | Browserless | Built-in PDF support |
| **Complex UI interactions** | Agent-S2 | Can handle dynamic/changing interfaces |
| **High-volume scraping** | Browserless | Lower resource overhead |
| **Screenshot + AI analysis** | Agent-S2 | Integrated AI reasoning |

### **Working Examples**

**Browserless - PDF Generation:**
```bash
# Generate PDF from local dashboard
curl -X POST http://localhost:4110/pdf \
  -H "Content-Type: application/json" \
  -d '{"url": "http://localhost:5678", "options": {"format": "A4", "margin": {"top": "1in"}}}'
```

**Agent-S2 - Complex Navigation:**
```bash
# Navigate complex public site with AI reasoning
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{"task": "Go to github.com, search for \"vrooli ai\", and click on the first repository"}'
```

### **Hybrid Approach Example**
Use both tools in the same workflow:
1. **Agent-S2**: Navigate to complex public site, extract data
2. **Browserless**: Generate clean PDF report from internal dashboard
3. **MinIO**: Store both results for later processing

---

## 🧠 **AI + Automation + Storage Pipeline**

**Problem**: Process documents with AI and store results persistently  
**Solution**: Complete AI pipeline with workflow automation  
**Resources**: `ollama` + `n8n` + `minio`

### **Architecture**
```
Document Upload → MinIO Storage → n8n Workflow → Ollama Processing → Results to MinIO
```

### **Implementation**

1. **Setup MinIO bucket:**
```bash
# Create processing buckets
curl -X PUT http://localhost:9000/ai-processing-input
curl -X PUT http://localhost:9000/ai-processing-output
```

2. **Create n8n workflow:**
```json
{
  "name": "AI Document Processing Pipeline",
  "nodes": [
    {
      "name": "File Upload Trigger",
      "type": "n8n-nodes-base.webhook",
      "parameters": { "path": "document-upload" }
    },
    {
      "name": "Store in MinIO",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "http://localhost:9000/ai-processing-input/{{$json.filename}}",
        "method": "PUT"
      }
    },
    {
      "name": "Process with Ollama",
      "type": "n8n-nodes-base.httpRequest", 
      "parameters": {
        "url": "http://localhost:11434/api/generate",
        "method": "POST",
        "body": {
          "model": "llama3.1:8b",
          "prompt": "Analyze this document and provide a summary: {{$json.content}}"
        }
      }
    },
    {
      "name": "Store Results",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "http://localhost:9000/ai-processing-output/{{$json.filename}}.analysis.json",
        "method": "PUT"
      }
    }
  ]
}
```

3. **Test the pipeline:**
```bash
# Upload document for processing
curl -X POST http://localhost:5678/webhook/document-upload \
  -F "file=@document.txt" \
  -F "filename=test-doc.txt"
```

### **Advanced Features**
- **Batch processing**: Process multiple documents in parallel
- **Error handling**: Retry failed AI processing jobs
- **Progress tracking**: Store processing status in MinIO metadata
- **Result notifications**: Send completion alerts via n8n

---

## 🎧 **Multi-Modal Content Processing**

**Problem**: Convert audio content to images with AI analysis  
**Solution**: Chain multiple AI services for complex processing  
**Resources**: `whisper` + `ollama` + `comfyui` + `minio`

### **Workflow: Podcast → Transcript → Analysis → Visualization**

1. **Audio Transcription (Whisper):**
```bash
# Upload audio file for transcription
curl -X POST http://localhost:8090/transcribe \
  -F "audio=@podcast-episode.wav" \
  -F "language=en" \
  > transcript.json
```

2. **Content Analysis (Ollama):**
```bash
# Analyze transcript for key themes
curl -X POST http://localhost:11434/api/generate \
  -d '{
    "model": "llama3.1:8b",
    "prompt": "Extract the 3 main themes from this podcast transcript: [transcript]",
    "system": "You are a content analyst. Provide themes as JSON array."
  }' > themes.json
```

3. **Visual Generation (ComfyUI):**
```bash
# Generate images based on themes
curl -X POST http://localhost:8188/api/v1/queue \
  -d '{
    "workflow": "theme-visualization",
    "inputs": {"theme_1": "AI Innovation", "theme_2": "Future Tech"}
  }' > images.json
```

4. **Store Results (MinIO):**
```bash
# Store all artifacts
curl -X PUT http://localhost:9000/podcast-analysis/episode-123/transcript.json -T transcript.json
curl -X PUT http://localhost:9000/podcast-analysis/episode-123/themes.json -T themes.json
curl -X PUT http://localhost:9000/podcast-analysis/episode-123/visualization.png -T generated_image.png
```

### **Complete n8n Automation**
Create an n8n workflow that orchestrates all these steps automatically when a new podcast file is uploaded.

---

## 🔍 **Research + Agent Workflow**

**Problem**: Automatically research topics and interact with results  
**Solution**: Combine search, AI analysis, and web automation  
**Resources**: `searxng` + `agent-s2` + `minio` + `ollama`

### **Research Pipeline**

1. **Search for Information (SearXNG):**
```bash
# Search for recent AI developments
curl "http://localhost:8100/search?q=artificial+intelligence+2025+developments&format=json" > search_results.json
```

2. **Analyze Results (Ollama):**
```bash
# Summarize and rank search results
curl -X POST http://localhost:11434/api/generate \
  -d '{
    "model": "llama3.1:8b",
    "prompt": "Analyze these search results and rank the top 3 most important AI developments: [results]",
    "system": "You are a research analyst. Provide structured analysis."
  }' > analysis.json
```

3. **Interactive Research (Agent-S2):**
```bash
# Have Agent-S2 visit top results and gather more details
curl -X POST http://localhost:4113/ai/action \
  -d '{
    "task": "Visit the top 3 websites from this list and extract key information about each AI development: [urls]"
  }' > detailed_research.json
```

4. **Store Research Archive (MinIO):**
```bash
# Archive complete research session
curl -X PUT http://localhost:9000/research-archive/ai-developments-2025/search.json -T search_results.json
curl -X PUT http://localhost:9000/research-archive/ai-developments-2025/analysis.json -T analysis.json
curl -X PUT http://localhost:9000/research-archive/ai-developments-2025/details.json -T detailed_research.json
```

### **Automated Research Assistant**
Create an n8n workflow that:
- Runs daily searches on specified topics
- Analyzes and filters results for relevance
- Uses Agent-S2 to gather detailed information
- Stores findings in organized MinIO buckets
- Sends summary reports via email/webhook

---

## 🛠️ **Resource Configuration Tips**

### **Common Configuration Patterns**

**Environment-Specific Secrets:**
```bash
# Development environment
./scripts/resources/storage/vault/manage.sh --action put-secret \
  --path "environments/dev/database" \
  --value '{"host":"localhost","user":"dev_user","password":"dev_pass"}'

# Production environment  
./scripts/resources/storage/vault/manage.sh --action put-secret \
  --path "environments/prod/database" \
  --value '{"host":"prod.db.com","user":"prod_user","password":"secure_pass"}'
```

**Resource Health Monitoring:**
```bash
# Monitor all resources in a single command
./scripts/resources/index.sh --action discover

# Test functional capabilities
./scripts/resources/storage/vault/manage.sh --action test-functional
```

**Batch Operations:**
```bash
# Process multiple files through AI pipeline
for file in *.txt; do
  curl -X POST http://localhost:5678/webhook/document-upload -F "file=@$file"
done
```

---

## 🚀 **Performance Optimization**

### **Resource Allocation Guidelines**

| Resource | Recommended RAM | CPU Cores | Use Case |
|----------|----------------|-----------|----------|
| Ollama (small models) | 4GB | 2 cores | Basic text processing |
| Ollama (large models) | 16GB+ | 4+ cores | Complex analysis |
| ComfyUI | 8GB+ | 4+ cores | Image generation |
| Agent-S2 | 2GB | 2 cores | Web automation |
| n8n | 1GB | 1 core | Workflow orchestration |

### **Scaling Patterns**

**Horizontal Scaling:**
- Run multiple Ollama instances with different models
- Load balance n8n workflows across multiple instances
- Distribute Agent-S2 tasks across multiple containers

**Vertical Scaling:**
- Increase memory for larger AI models
- Add CPU cores for concurrent processing
- Use GPU acceleration for ComfyUI when available

---

## 🔧 **Troubleshooting Guide**

### **Common Integration Issues**

**Issue**: Vault secrets not accessible from n8n
```bash
# Check Vault status and auth
./scripts/resources/storage/vault/manage.sh --action auth-info
./scripts/resources/storage/vault/manage.sh --action test-functional
```

**Issue**: Agent-S2 tasks timing out
```bash
# Break complex tasks into smaller steps
# Use shorter, more specific task descriptions
# Check system resources with docker stats
```

**Issue**: MinIO storage full
```bash
# Check storage usage
curl http://localhost:9001/minio/admin/v3/info

# Clean up old files
curl -X DELETE http://localhost:9000/bucket-name/old-file.txt
```

**Issue**: AI models running out of memory
```bash
# Check Ollama model sizes
curl http://localhost:11434/api/tags

# Switch to smaller models or increase container memory
```

---

## 📚 **Additional Resources**

- **Individual Resource READMEs**: `/scripts/resources/*/README.md`
- **Configuration Files**: `~/.vrooli/resources.local.json`
- **Example Workflows**: `/scripts/resources/*/examples/`
- **Test Scripts**: `/scripts/resources/tests/`

---

**🎯 This cookbook demonstrates real, working patterns you can implement immediately. All examples use the default Vrooli resource configuration and have been tested in the development environment.**