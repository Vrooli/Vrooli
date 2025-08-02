# Resource Relationship Matrix

## Resource Co-occurrence Matrix

This matrix shows how often resources are used together in test scenarios.

```
                    a  b  c  o  u  w  c  h  n  n  w  m  p  q  q  r  v  s  j
                    g  r  l  l  n  h  o  u  8  o  i  i  o  d  u  e  a  e  u
                    e  o  a  l  s  i  m  g  n  d  n  n  s  r  e  d  u  a  d
                    n  w  u  a  t  s  f  i     e  d  i  t  a  s  i  l  r  g
                    t  s  d  m  r  p  y  n     -  m  o  g  n  t  s  t  x  e
                    -  e  e  a  u  e  u  n     r  i     r  t  d     n  0
                    s  r  -     c  r  i        e  l     e     b     g
                    2  l  c     t     
                       e  o     u
                       s  d     r
                       s  e     e
                                -
                                i
                                o

agent-s2            -  2  0  4  0  2  2  0  1  0  0  1  0  1  0  0  0  0  0
browserless         2  -  0  1  0  1  1  0  0  1  0  1  0  0  1  0  0  0  0
claude-code         0  0  -  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0
ollama              4  1  0  -  4  4  2  0  1  0  0  1  0  5  0  0  0  1  0
unstructured-io     0  0  0  4  -  1  0  0  0  0  0  2  1  2  0  0  1  0  0
whisper             2  1  0  4  1  -  2  0  0  0  0  1  0  1  0  0  0  0  0
comfyui             2  1  0  2  0  2  -  0  0  0  0  1  0  1  0  0  0  0  0
huginn              0  0  0  0  0  0  0  -  0  0  0  0  0  0  0  0  0  0  0
n8n                 1  0  0  1  0  0  0  0  -  0  0  0  0  0  0  0  1  0  0
node-red            0  1  0  0  0  0  0  0  0  -  0  0  0  0  1  0  0  0  0
windmill            0  0  0  0  0  0  0  0  0  0  -  0  0  0  0  0  0  0  0
minio               1  1  0  1  2  1  1  0  0  0  0  -  1  1  0  0  1  0  0
postgres            0  0  0  0  1  0  0  0  0  0  0  1  -  0  0  0  1  0  0
qdrant              1  0  0  5  2  1  1  0  0  0  0  1  0  -  0  0  0  1  0
questdb             0  1  0  0  0  0  0  0  0  1  0  0  0  0  -  0  0  0  0
redis               0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  -  0  0  0
vault               0  0  0  0  1  0  0  0  1  0  0  1  1  0  0  0  -  0  0
searxng             0  0  0  1  0  0  0  0  0  0  0  0  0  1  0  0  0  -  0
judge0              0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  -
```

## Key Insights from Co-occurrence

### Strong Partnerships (3+ co-occurrences)
1. **ollama + qdrant** (5) - Vector search for AI responses
2. **ollama + unstructured-io** (4) - Document AI processing
3. **ollama + whisper** (4) - Multimodal AI (audio + text)
4. **ollama + agent-s2** (4) - AI-driven automation

### Resource Hubs (most connections)
1. **ollama** - Connected to 8 other resources (hub for AI workflows)
2. **minio** - Connected to 7 other resources (central storage)
3. **qdrant** - Connected to 6 other resources (vector search backbone)

### Isolated Resources (0 connections)
- claude-code
- huginn
- windmill
- redis
- judge0

## Resource Categories and Their Interactions

### AI Processing Core
```
whisper ←→ ollama ←→ qdrant
   ↓         ↓         ↑
   └──→ unstructured-io ┘
```

### Automation Layer
```
agent-s2 ←→ browserless
    ↓           ↓
    └──→ ollama ←┘
```

### Storage Backend
```
minio ←→ vault
  ↓        ↓
postgres ←─┘
```

### Workflow Orchestration
```
n8n ←→ vault
node-red ←→ questdb
```

## Recommended Resource Combinations for New Tests

Based on the analysis, here are optimal resource combinations for comprehensive testing:

### 1. Full AI Pipeline
```yaml
resources: [ollama, whisper, unstructured-io, qdrant, minio, redis]
purpose: Complete AI processing with caching
```

### 2. Secure Automation
```yaml
resources: [agent-s2, vault, judge0, postgres]
purpose: Secure code execution with secrets management
```

### 3. Event-Driven Analytics
```yaml
resources: [huginn, questdb, redis, ollama]
purpose: Real-time event processing with AI insights
```

### 4. Development Platform
```yaml
resources: [windmill, claude-code, judge0, postgres, redis]
purpose: Full development environment with AI assistance
```

### 5. Search and Discovery
```yaml
resources: [searxng, browserless, qdrant, ollama]
purpose: Intelligent web search and knowledge extraction
```