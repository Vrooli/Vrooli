# Vrooli Context and Vision

## What Vrooli Is

Vrooli is a **self-improving AI platform** that orchestrates local resources to build complete applications. It represents a new paradigm where:

- **AI agents are the primary developers**, not humans
- **Every scenario becomes a permanent capability** that enhances the system
- **Scenarios are practical business applications** that solve real problems
- **The platform combines 30+ local resources** into powerful solutions
- **Knowledge is permanent** through the Qdrant memory system

## Core Architecture

### Resources (Building Blocks)
Resources are the fundamental services that provide capabilities:
- **AI/ML**: claude-code, ollama, comfyui, whisper
- **Storage**: postgres, redis, minio, qdrant, questdb
- **Development**: judge0, browserless, vault
- **Monitoring**: Various monitoring and security tools

### Scenarios (Applications)
Scenarios are complete business applications built using resources:
- Each scenario solves a specific business problem
- Scenarios can leverage any combination of resources
- They include API, CLI, and UI components
- Scenarios generate revenue through real-world usage

### Memory System (Qdrant)
The Qdrant vector database serves as Vrooli's long-term memory:
- **Every solution, pattern, and failure is remembered**
- **All agents can search and learn from past work**
- **Knowledge compounds over time**
- **No work is ever repeated**

## Development Philosophy

### AI-First Development
- AI agents (primarily Claude Code) do the actual development
- Humans provide high-level guidance and requirements
- The system learns from every iteration

### PRD-Driven Approach
- Every scenario/resource has a Product Requirements Document (PRD)
- PRDs define the permanent capability being added
- Progress is tracked through PRD completion
- Requirements are prioritized: P0 (must) → P1 (should) → P2 (nice)

### Continuous Improvement
- Generator scenarios create new capabilities (one-time seeding)
- Improver scenarios iteratively enhance existing capabilities
- The system becomes more capable with each iteration

## Business Model

### Value Creation
- Each scenario solves specific business problems
- Scenarios provide measurable efficiency gains
- The platform can generate hundreds of scenarios
- Total platform value compounds over time

### Deployment Models
- Direct scenario deployment
- SaaS access for scenario usage
- Custom scenario development
- Enterprise installations

## Key Principles

1. **Memory First**: Always search Qdrant before starting work
2. **No Repeated Work**: Learn from what exists
3. **Incremental Progress**: Small, validated improvements
4. **Cross-Scenario Synergy**: Scenarios enhance each other
5. **Quality Over Quantity**: Better to do one thing well
6. **Business Value Focus**: Every scenario must generate revenue


## Remember

You are building permanent capabilities that will:
- Generate real revenue
- Solve real problems
- Become part of Vrooli's collective intelligence
- Enable future scenarios to be even more powerful

Every line of code you write becomes part of Vrooli's DNA forever.