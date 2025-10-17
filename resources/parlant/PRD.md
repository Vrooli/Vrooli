# Parlant LLM Agent Framework PRD

## Executive Summary
**What**: Production-ready Python SDK for controlled multi-agent development with behavioral guidelines
**Why**: Ensures predictable agent behavior through policy enforcement, not hope-based compliance  
**Who**: Scenarios requiring reliable customer service automation and enterprise chatbots
**Value**: $100K+ from enterprise deployments requiring consistent, auditable agent behavior
**Priority**: P0 - Essential for production AI agent deployments

## P0 Requirements (Must Have)
- [x] **v2.0 Contract Compliance**: Full implementation of universal.yaml requirements
- [x] **Health Check Endpoint**: HTTP endpoint responding within 5 seconds
- [x] **Python SDK Server**: FastAPI server hosting Parlant SDK capabilities
- [x] **Behavioral Guidelines**: Natural language behavior control with contextual matching
- [x] **Journey Definitions**: Multi-step customer interaction flow management
- [x] **Tool Registration**: Decorator-based tool integration system
- [x] **Context Variables**: Dynamic context management for agent state

## P1 Requirements (Should Have)
- [x] **Self-Critique Engine**: Real-time guideline adherence validation
- [x] **Guideline Conflict Detection**: Alert on contradictory instructions
- [x] **Audit Logging**: Decision trail for compliance and debugging
- [x] **Response Templates**: Canned responses for consistency

## P2 Requirements (Nice to Have)
- [ ] **Multi-Model Support**: Integration with Claude, GPT-4, Ollama
- [ ] **Webhook Integration**: External event-driven tool execution

## Technical Specifications

### Architecture
- **Port**: 11458 (AI services range, next available)
- **Dependencies**: Python 3.11+, FastAPI, Pydantic
- **Storage**: PostgreSQL for conversation history, Redis for session state
- **API Style**: REST with WebSocket support for streaming

### Core Components
1. **Policy Layer**: Guidelines and journeys as declarative objects
2. **Tool Layer**: External service integrations via decorators
3. **Inference Layer**: LLM model abstraction supporting multiple providers

### API Endpoints
- `POST /agents` - Create new agent with configuration
- `POST /agents/{id}/guidelines` - Add behavioral guidelines
- `POST /agents/{id}/journeys` - Define customer journeys
- `POST /agents/{id}/tools` - Register tools/functions
- `POST /agents/{id}/chat` - Send message to agent
- `GET /agents/{id}/history` - Retrieve conversation history
- `GET /health` - Health check endpoint

### CLI Commands (v2.0 compliant)
- `vrooli resource parlant help` - Show comprehensive help
- `vrooli resource parlant info` - Show runtime configuration
- `vrooli resource parlant manage [install|start|stop|restart|uninstall]` - Lifecycle
- `vrooli resource parlant test [smoke|integration|unit|all]` - Testing
- `vrooli resource parlant content [create-agent|list-agents|add-guideline|add-journey]` - Agent management
- `vrooli resource parlant status` - Show service status
- `vrooli resource parlant logs` - View service logs

## Success Metrics

### Completion Targets
- **P0 Completion**: 100% (7/7 requirements)
- **P1 Completion**: 75% (3/4 requirements)
- **P2 Completion**: 50% (1/2 requirements)

### Quality Metrics
- **Test Coverage**: >80% for core functionality
- **Response Time**: <500ms for guideline matching
- **Startup Time**: <30 seconds
- **Memory Usage**: <512MB baseline

### Performance Targets
- **Concurrent Agents**: Support 50+ active agents
- **Guidelines per Agent**: Handle 100+ guidelines efficiently
- **Message Throughput**: 100 messages/second

## Research Findings

### Similar Work
- **AutoGPT**: Autonomous task execution but lacks behavioral control
- **CrewAI**: Multi-agent orchestration without journey definitions  
- **AutoGen Studio**: Agent development but missing production controls

### Template Selected
Using existing Python-based resources (langchain, pandas-ai) as structural templates

### Unique Value
Parlant uniquely combines declarative policy control with production reliability features, ensuring agents follow business rules through enforcement rather than prompting

### External References
- https://github.com/emcie-co/parlant
- https://pypi.org/project/parlant/
- https://www.parlant.io/
- Apache 2.0 licensed open source

### Security Notes
- API key management through environment variables
- Session isolation between agents
- Audit logging for compliance
- No direct LLM prompt injection vulnerabilities

## Implementation Progress

### History
- 2025-01-10: Initial PRD creation, research completed
- 2025-01-10: v2.0 structure implemented with core.sh, test.sh, and lifecycle management
- 2025-01-10: Health check endpoint and FastAPI server implemented
- 2025-01-10: Basic agent creation and guideline management working
- 2025-01-10: Smoke tests passing (4/4 tests)
- 2025-09-11: Completed all P0 requirements:
  - Enhanced journey definitions with multi-step flow management
  - Implemented tool registration with webhook/python/external support
  - Added comprehensive context variable management with sessions
  - Updated chat endpoint with journey and context support
- 2025-09-11: Completed all P1 requirements:
  - Implemented self-critique engine with adherence scoring
  - Added guideline conflict detection with detailed warnings
  - Created comprehensive audit logging system with filtering
  - Built response template management with variable substitution
  - Enhanced test suite to validate P1 features

### Current Completion
- **P0 Requirements**: 100% (7/7 completed) ✅
- **P1 Requirements**: 100% (4/4 completed) ✅
- **P2 Requirements**: 0% (0/2 completed)
- **Overall**: ~85% functional