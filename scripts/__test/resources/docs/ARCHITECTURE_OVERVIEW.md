# 🏗️ Vrooli Resource Test Framework - Architecture Overview

**Visual guide to understanding how everything fits together**

## 🎯 The Big Picture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           🚀 ./run.sh (Your Entry Point)                    │
└───────────────────────────┬─────────────────────────────────────────────────┘
                            │
            ┌───────────────┴───────────────┐
            │                               │
    ┌───────▼───────┐                ┌─────▼─────┐
    │ 🔍 Discovery   │                │ 🏃 Runner │
    │ What's running?│                │ Run tests │
    └───────┬───────┘                └─────┬─────┘
            │                               │
    ┌───────▼───────┐                ┌─────▼─────┐
    │ 📊 Reporter    │                │ 🧪 Tests │
    │ Show results   │                │ Your code │
    └───────────────┘                └───────────┘
```

## 🗂️ File Structure (What Goes Where)

```
tests/
├── 🚀 run.sh                          # ← Your main entry point
├── 📖 GETTING_STARTED.md              # ← Start here for quick setup
├── 🏗️ ARCHITECTURE_OVERVIEW.md        # ← You are here
│
├── 🔧 framework/                      # The engine (usually don't modify)
│   ├── discovery.sh                   # Finds running services
│   ├── runner.sh                      # Executes tests with isolation
│   ├── reporter.sh                    # Formats test results
│   ├── interface-compliance.sh        # Validates service interfaces
│   ├── capability-registry.sh         # Validates service capabilities
│   ├── integration-patterns.sh        # Tests multi-service workflows
│   ├── performance-benchmarks.sh      # Measures performance
│   └── helpers/                       # Utility functions
│
├── 🎯 single/                         # Individual service tests
│   ├── ai/
│   │   ├── ollama.test.sh             # ← Example of a great test
│   │   ├── whisper.test.sh
│   │   └── unstructured-io.test.sh
│   ├── automation/
│   │   ├── n8n.test.sh
│   │   ├── node-red.test.sh
│   │   └── windmill.test.sh
│   ├── agents/
│   │   ├── agent-s2.test.sh
│   │   └── browserless.test.sh
│   └── storage/
│       ├── minio.test.sh
│       ├── postgres.test.sh
│       └── redis.test.sh
│
├── 🎭 scenarios/                      # Multi-service business workflows
│   ├── multi-modal-ai-assistant/      # ← Example of complete workflow
│   ├── document-intelligence-pipeline/
│   ├── research-assistant/
│   └── business-process-automation/
│
└── 🧪 fixtures/                       # Test data
    ├── audio/                         # Sample audio files
    ├── documents/                     # Sample documents
    ├── images/                        # Sample images
    └── workflows/                     # Sample workflow configs
```

## 🔄 Test Execution Flow

```
1. 🎬 DISCOVERY PHASE
   │
   ├── run.sh starts
   ├── Scans for running services
   ├── Validates service health
   └── Creates list of testable resources
   
2. 🏃 EXECUTION PHASE
   │
   ├── For each test:
   │   ├── Creates isolated environment
   │   ├── Sets up cleanup handlers
   │   ├── Runs actual test code
   │   ├── Captures results
   │   └── Cleans up environment
   │
   └── Collects all results
   
3. 📊 REPORTING PHASE
   │
   ├── Aggregates test results
   ├── Generates summary statistics
   ├── Provides helpful error messages
   └── Outputs in requested format (text/JSON)
```

## 🧩 Component Relationships

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          SERVICE DISCOVERY                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                │
│  │   Ollama    │    │     n8n     │    │  Agent-S2   │                │
│  │ :11434      │    │ :5678       │    │ :4113       │                │
│  └─────────────┘    └─────────────┘    └─────────────┘                │
└──────────────────────┬─────────────────────────────────────────────────┘
                       │
┌──────────────────────▼─────────────────────────────────────────────────┐
│                      TEST FRAMEWORK                                    │
│                                                                        │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐              │
│  │ Single Tests  │  │  Integration  │  │   Business    │              │
│  │               │  │   Patterns    │  │  Scenarios    │              │
│  │ • Basic health│  │               │  │               │              │
│  │ • API calls   │  │ • AI+Storage  │  │ • Multi-modal │              │
│  │ • Performance │  │ • Auto+Store  │  │ • Document AI │              │
│  └───────────────┘  └───────────────┘  └───────────────┘              │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                     VALIDATION LAYERS                            │  │
│  │                                                                  │  │
│  │  Interface      Capability      Performance     Integration    │  │
│  │  Compliance  →  Registry     →  Benchmarks   →  Patterns       │  │
│  │                                                                  │  │
│  │  "Does it       "Can it do      "Is it fast    "Do they work   │  │
│  │   follow        what we         enough?"        together?"      │  │
│  │   standards?"   need?"                                          │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

## 🎯 Which File Should I Modify?

**Use this decision tree to find the right file:**

```
❓ What do you want to do?

├── 🧪 "Test a new service I added"
│   └── 📁 single/CATEGORY/your-service.test.sh
│       └── 📋 Copy pattern from single/ai/ollama.test.sh
│
├── 🔗 "Test multiple services working together"
│   └── 📁 scenarios/your-workflow/test.sh
│       └── 📋 Copy pattern from scenarios/multi-modal-ai-assistant/
│
├── ⚙️ "Change how tests run"
│   └── 📄 framework/runner.sh
│
├── 🔍 "Change how services are discovered"
│   └── 📄 framework/discovery.sh
│
├── 📊 "Change test output format"
│   └── 📄 framework/reporter.sh
│
├── 📈 "Add performance benchmarks"
│   └── 📄 framework/performance-benchmarks.sh
│
├── 🎛️ "Add new command-line options"
│   └── 📄 run.sh
│
└── 📖 "Improve documentation"
    └── 📄 Any .md file or create new one
```

## 🧠 Key Concepts

### **Resource Categories**
```
🧠 AI Resources      → ollama, whisper, comfyui
⚙️ Automation       → n8n, node-red, windmill
🤖 Agents           → agent-s2, browserless, claude-code  
🔍 Search           → searxng
💾 Storage          → minio, postgres, redis, qdrant
⚡ Execution        → judge0
```

### **Test Types**
```
🎯 Single Resource Tests
   ├── Health checks
   ├── Basic functionality  
   ├── Error handling
   └── Performance

🔗 Integration Tests
   ├── AI + Storage
   ├── Automation + Storage
   ├── AI + Automation
   └── Multi-resource pipelines

🎭 Business Scenarios
   ├── Complete workflows
   ├── Revenue-generating features
   ├── Client-ready solutions
   └── Market-validated use cases
```

### **Validation Layers**
```
Layer 1: Interface Compliance
   └── "Does it implement the standard actions?"

Layer 2: Capability Registry  
   └── "Can it do what its category requires?"

Layer 3: Performance Benchmarks
   └── "Is it fast enough for production?"

Layer 4: Integration Patterns
   └── "Does it work well with other services?"
```

## 🛠️ Development Workflow

```
1. 📋 PLANNING
   ├── Identify what you want to test
   ├── Choose the right file/directory
   └── Look at similar existing examples

2. 🔨 DEVELOPMENT  
   ├── Copy existing pattern
   ├── Modify for your specific case
   ├── Add proper error handling
   └── Include helpful output messages

3. 🧪 TESTING
   ├── Run your specific test: ./run.sh --resource yours
   ├── Run with debug: ./run.sh --resource yours --debug
   └── Verify it works in CI: ./run.sh --output-format json

4. 📚 DOCUMENTATION
   ├── Update relevant README files
   ├── Add examples to COMMON_PATTERNS.md
   └── Include troubleshooting tips
```

## 💡 Pro Tips

1. **Follow Existing Patterns**: The `ollama.test.sh` file is the gold standard
2. **Use Categories**: Put tests in the right category directory (ai/, automation/, etc.)
3. **Test Real Services**: Never mock - always test against actual running services
4. **Include Business Value**: Scenarios should represent real client work
5. **Make Tests Helpful**: Include clear error messages and suggestions
6. **Think About CI/CD**: Tests should work in automated environments

---

**🎉 Now You Understand the Architecture!** 

Use this as a reference when working with the framework. The key is that everything follows predictable patterns - once you understand one part, the rest follows the same logic.

**Next Steps**: Check out [GETTING_STARTED.md](GETTING_STARTED.md) for hands-on examples!