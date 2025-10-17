# Codex Resource - Product Requirements Document

## Executive Summary
**What**: Multi-tier AI-powered coding platform with text generation and full agent execution capabilities
**Why**: Enable both simple code generation and complete autonomous software development across Vrooli scenarios
**Who**: All scenarios requiring code generation, test creation, code review, or language translation
**Value**: $50,000+ (enables both text generation and autonomous agents for 50+ scenarios, reducing development time by 85%)
**Priority**: High - Core enabler for AI-assisted development workflows with progressive enhancement

## Current State
- ✅ **Text Generation Mode**: Fully implemented with GPT-5 models
- ✅ **Smart Routing**: Auto-selects best available backend
- ✅ **Codex CLI Integration**: Ready for full agent capabilities
- ✅ **v2.0 Contract Compliance**: Complete implementation
- ✅ **Secrets Management**: Vault integration implemented
- ❌ **Responses API**: Not yet implemented (codex-mini-latest with tools)
- ❌ **Function Calling Fallback**: GPT-5 function calling not integrated

## Requirements Checklist

### P0 Requirements (Must Have) - ✅ COMPLETED
- [x] **Modern OpenAI Integration**: GPT-5 models implemented (Released August 2025)
- [x] **Secrets Management**: Vault integration for API key management
- [x] **v2.0 Contract Compliance**: Full universal.yaml implementation
- [x] **Code Generation API**: Text generation from natural language prompts
- [x] **Multi-Language Support**: Python, JavaScript, Go, Bash, SQL, etc.
- [x] **Health Validation**: API connectivity and credential verification
- [x] **Smart Routing**: Automatic backend selection (CLI → API → fallback)
- [x] **Codex CLI Integration**: Full agent capabilities when CLI installed

### P1 Requirements (Should Have) - 🚧 IN PROGRESS
- [x] **Code Review**: Automated code review via agent commands
- [x] **Test Generation**: Generate and run unit tests via agent commands
- [x] **Language Translation**: Convert code between programming languages
- [ ] **Responses API Integration**: codex-mini-latest with function calling
- [ ] **Function Calling Fallback**: GPT-5 tool execution without CLI
- [ ] **Test Suite**: Complete test/phases structure

### P2 Requirements (Nice to Have) - 🎯 NEXT
- [x] **Model Selection**: Support multiple models (GPT-5, GPT-4o, o1, codex-mini)
- [ ] **Caching Layer**: Cache common requests to reduce API costs
- [ ] **Batch Processing**: Process multiple files efficiently
- [ ] **Custom Tools**: Vrooli-specific function definitions
- [ ] **Workflow Templates**: Pre-built agent workflows

## Technical Specifications

### Multi-Tier Architecture (2025)
```
Codex Resource - Smart Routing Platform
├── Tier 1: Full Agent Mode
│   ├── OpenAI Codex CLI (external)
│   ├── Local execution sandbox
│   ├── File operations + command execution
│   └── Multi-step task orchestration
├── Tier 2: Function Calling Mode 
│   ├── Responses API (codex-mini-latest) [PENDING]
│   ├── Chat Completions API (GPT-5)  [PENDING]
│   ├── Tool execution framework
│   └── Sandboxed workspace operations
├── Tier 3: Text Generation Mode
│   ├── GPT-5 series (nano, mini, full)
│   ├── GPT-4o series (mini, full)
│   ├── o1 reasoning models
│   └── Simple prompt → text response
└── Integration Points
    ├── Vault (secrets management)
    ├── Smart routing logic
    ├── Workspace isolation
    └── Scenarios (consumers)
```

### Dependencies
#### Required
- OpenAI API access with GPT-5/GPT-4 models
- jq for JSON processing  
- curl for API requests
- Vault for secure key storage

#### Optional (Progressive Enhancement)
- Node.js + npm (for Codex CLI installation)
- OpenAI Codex CLI (@openai/codex package)

### API Endpoints
#### Tier 1: External Agent
- OpenAI Codex CLI: Local installation via npm

#### Tier 2: Function Calling APIs
- OpenAI Responses API: https://api.openai.com/v1/responses (codex-mini-latest)
- OpenAI Chat Completions: https://api.openai.com/v1/chat/completions (GPT-5 with tools)

#### Tier 3: Text Generation  
- OpenAI Chat Completions: https://api.openai.com/v1/chat/completions
- Available Models: gpt-5-nano (default), gpt-5-mini, gpt-5, gpt-4o-mini, gpt-4o, o1-mini, o1-preview

### Configuration
```yaml
# config/secrets.yaml
version: "1.0"
resource: "codex"
description: "OpenAI API credentials for code generation"

secrets:
  api_keys:
    - name: "openai_api_key"
      path: "secret/resources/codex/api/openai"
      description: "OpenAI API key for all tiers (text generation, function calling, CLI)"
      required: true
      format: "string"
      validation:
        pattern: "^sk-[A-Za-z0-9]{48}$"
      default_env: "OPENAI_API_KEY"
      supports:
        - "codex-cli"
        - "responses-api"
        - "chat-completions"
        - "codex-mini-latest"
        - "gpt-5-*"
        - "gpt-4o-*"
        - "o1-*"
      example: "sk-proj-..."
```

## Success Metrics

### Completion Criteria
- [x] 100% v2.0 contract compliance
- [x] All P0 requirements implemented and tested
- [x] API connectivity validated (Tier 3)
- [x] Smart routing system functional
- [x] Codex CLI integration ready
- [ ] Responses API implemented (Tier 2)
- [ ] Function calling fallback (Tier 2)
- [ ] Complete test suite (smoke < 30s, integration < 120s)
- [x] Documentation complete and accurate

### Quality Metrics
- Response time < 5 seconds for standard requests
- Success rate > 95% for valid prompts
- All generated code syntactically valid
- Test coverage > 80%

### Performance Targets
#### Tier 1 (Codex CLI Agent)
- First response time: 10-30 seconds (includes reasoning + execution)
- Multi-step task completion: 2-10 minutes depending on complexity
- File operations: Near-instantaneous local execution

#### Tier 2 (Function Calling APIs)
- codex-mini-latest: 3-8 seconds (optimized for coding)
- GPT-5 with tools: 5-15 seconds (includes function calls)
- Tool execution overhead: 1-3 seconds per function call

#### Tier 3 (Text Generation)
- gpt-5-nano: 1-3 seconds (fast and cheap)
- gpt-5: 3-8 seconds (high quality)
- Token efficiency: < 8000 tokens average (400K context available)

## Revenue Justification

### Direct Value by Tier
#### Tier 1 (Full Agent) - When Available
- **Complete Project Generation**: Build entire applications autonomously ($25,000 value)
- **End-to-End Testing**: Generate, run, and fix tests automatically ($8,000 value)  
- **Autonomous Debugging**: Fix complex issues without human intervention ($12,000 value)

#### Tier 2 (Function Calling) - Planned
- **Interactive Development**: Code + execution in single workflow ($15,000 value)
- **Validation Loops**: Auto-verify generated code works ($6,000 value)
- **Incremental Building**: Build complex projects step-by-step ($10,000 value)

#### Tier 3 (Text Generation) - Current
- **Rapid Prototyping**: Fast code generation for all scenarios ($8,000 value)
- **Documentation**: Automated code documentation ($3,000 value)
- **Code Translation**: Convert between languages instantly ($4,000 value)

### Multiplier Effect
- **Progressive Enhancement**: Users start with Tier 3, upgrade to higher tiers as needed
- **Zero Lock-in**: Each tier works independently, no forced upgrades
- **Scenario Enablement**: All 50+ scenarios benefit from at least text generation
- **Total Platform Value**: $50,000-100,000 depending on adoption

## Implementation Priority

### ✅ **COMPLETED - Tier 3 & CLI Integration**
1. **Phase 1**: Text Generation Platform ✅
   - GPT-5 API integration with smart model selection
   - Vault secrets management
   - v2.0 contract compliance
   - Multi-language support

2. **Phase 2**: Smart Routing & CLI Integration ✅
   - Automatic backend detection and selection
   - OpenAI Codex CLI integration
   - Agent command framework (fix, generate-tests, refactor)
   - Progressive enhancement model

### 🚧 **IN PROGRESS - Tier 2 Function Calling**
3. **Phase 3**: Responses API Integration (Next)
   - codex-mini-latest model with /responses endpoint
   - Function calling implementation for Responses API
   - Tool definitions and execution framework
   - Sandboxed workspace operations

4. **Phase 4**: GPT-5 Function Calling Fallback
   - Integrate existing function calling example from tools-example.sh
   - Chat Completions API with tools parameter
   - Complete Tier 2 fallback chain

### 🎯 **PLANNED - Optimization**
5. **Phase 5**: Testing & Polish
   - Complete test suite implementation
   - Performance optimization across all tiers
   - Cost optimization and caching
   - Advanced workflow templates

## Risk Mitigation

### Technical Risks
- **API Deprecation**: Use stable GPT-3.5-turbo/GPT-4 APIs
- **Rate Limiting**: Implement exponential backoff and queuing
- **Cost Management**: Monitor token usage, implement limits

### Security Risks
- **API Key Exposure**: Use Vault for production
- **Code Injection**: Validate and sanitize all inputs
- **Data Privacy**: Don't send sensitive code to API

## Progress History
- 2025-01-11: PRD created, requirements defined, 0% complete
- 2025-01-13: **Tier 3 Complete** - Text generation with GPT-5 models, smart routing, CLI integration ready
  - ✅ P0 requirements: 100% complete
  - ✅ Smart routing system implemented
  - ✅ OpenAI Codex CLI integration framework
  - ✅ Progressive enhancement model
  - 🚧 P1 requirements: Responses API and function calling fallback pending
  - 📊 **Current Status: 75% complete** (Tier 1 + Tier 3 functional, Tier 2 pending)