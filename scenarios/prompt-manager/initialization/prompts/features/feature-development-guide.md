# Vrooli Feature Development Guide

You need to develop or fix something in Vrooli, but you're not sure which specific prompt to use. This guide helps you identify the right development prompt based on what you're trying to accomplish.

## ⚠️ Important: Working Directory

**All commands and paths in these feature development prompts assume you are working from the Vrooli project root directory.** 

Before running any commands, ensure you are in `/path/to/your/Vrooli/` directory. All relative paths like `scripts/resources/` are relative to this root directory, not to individual scenario directories or other subdirectories.

If you're working in a different directory, either:
1. `cd` to the Vrooli project root, or  
2. Adjust the paths accordingly (e.g., use `../../scripts/resources/` if you're two levels deep)

## Quick Decision Tree

### 🎯 What are you trying to build or fix?

**1. Adding a New Service or Tool** → Use `add-fix-resource.md`
- Examples: Ollama model integration, PostgreSQL setup, ComfyUI workflows, Whisper transcription
- If it extends Vrooli's capabilities through a new local service

**2. Creating Automation or Workflows** → Use `add-fix-n8n-workflow.md`  
- Examples: Claude reasoning chains, document processing pipelines, resource health monitoring
- If it involves connecting services or automating processes

**3. Building Complete Applications** → Use `add-fix-scenario.md`
- Examples: Prompt Manager, Audio Intelligence Platform, Brand Manager scenarios
- If it's a full application that uses multiple resources together

**4. Improving Platform Infrastructure** → Use `add-fix-core-features.md`
- Examples: CLI commands, system utilities, configuration management, testing framework
- If it's foundational code that other components depend on

**5. Not Sure Which Category** → Continue reading this guide

## Detailed Classification

### 🔧 Resources (Local Services)
**Choose `add-fix-resource.md` if you're working on:**

**AI & Processing Services:**
- New AI models or inference engines
- Document processing services  
- Image/audio/video processing tools
- Data analysis or ML services

**Automation Platforms:**
- Workflow engines (beyond n8n)
- Task schedulers
- Event processing systems
- Integration platforms

**Storage & Databases:**
- New database systems
- File storage solutions
- Cache systems
- Search engines

**Development Tools:**
- Code execution environments
- Testing frameworks
- Monitoring tools
- Security services

**Signs you need the resource prompt:**
- It runs as a separate service/process
- Other scenarios will benefit from using it
- It has its own API or interface
- It needs port allocation and lifecycle management

### 🔄 N8n Workflows (Automation Logic)
**Choose `add-fix-n8n-workflow.md` if you're working on:**

**Data Processing Workflows:**
- ETL pipelines
- Data transformation chains
- API data synchronization
- Report generation automation

**Business Logic Automation:**
- Multi-step decision processes
- Approval workflows
- Notification systems
- Background task processing

**Resource Orchestration:**
- Coordinating multiple Vrooli resources
- Complex multi-step operations
- Error handling and retry logic
- Stateful process management

**Signs you need the n8n workflow prompt:**
- It's primarily about connecting existing services
- It involves conditional logic or decision trees
- It processes data through multiple steps
- It automates business processes

### 🏗️ Scenarios (Complete Applications)
**Choose `add-fix-scenario.md` if you're working on:**

**SaaS Applications:**
- Customer-facing business applications
- Revenue-generating solutions ($10K-50K value)
- Complete user experiences with UI/API/CLI
- Multi-tenant or scalable applications

**Business Solutions:**
- Industry-specific tools
- Productivity applications
- Data analysis platforms
- Content management systems

**Integration Applications:**
- Apps that connect multiple external services
- Migration or synchronization tools
- Monitoring and reporting dashboards
- Testing and validation suites

**Signs you need the scenario prompt:**
- It's a complete application users interact with
- It has business value as a standalone product  
- It combines multiple resources to solve a specific problem
- It needs its own database, API, and user interface

### ⚙️ Core Features (Platform Infrastructure)
**Choose `add-fix-core-features.md` if you're working on:**

**CLI & User Interface:**
- New vrooli commands
- Command-line tools and utilities
- Help systems and documentation
- User experience improvements

**System Management:**
- Lifecycle management (setup, develop, build, deploy)
- Configuration management
- Environment handling
- Service orchestration

**Shared Libraries:**
- Common utilities used across the platform
- Logging and monitoring systems
- Security and authentication
- Performance optimization tools

**Developer Infrastructure:**
- Testing frameworks
- Development tools
- Debugging utilities
- Code generation tools

**Signs you need the core features prompt:**
- It's infrastructure that other components depend on
- It's part of the vrooli CLI or main management system
- It's shared utilities or libraries
- It improves the developer experience

## Common Multi-Prompt Projects

Some projects require multiple prompts in sequence:

• **AI-Powered App**: Resource → Workflows → Scenario (e.g., Audio Intelligence Platform)
• **Platform Enhancement**: Core Features → Resource Updates → Scenario Updates  
• **Business Solution**: Workflows → Scenario → Core Features (as needed)
• **New Resource Integration**: Resource → Workflows → Update existing scenarios

## How Each Prompt Connects to Vrooli's Architecture

### 🎯 Self-Improving Intelligence Loop

**Resources** → Expand what Vrooli can do
- Each new resource multiplies the capabilities of ALL scenarios
- Resources are building blocks that enable new types of applications

**N8n Workflows** → Make scenarios smarter and more autonomous  
- Workflows handle complex logic so scenarios don't need custom code
- Reusable workflows make building new scenarios faster

**Scenarios** → Turn capabilities into business value
- Each scenario becomes a permanent asset worth $10K-50K
- Scenarios prove that resource combinations work and create value

**Core Features** → Make everything more robust and reliable
- Better platform = more reliable scenarios = happier customers
- Developer experience improvements accelerate scenario development

### 🔄 The Recursive Enhancement Cycle

1. **Build Better Core Features** → Easier to create resources and scenarios
2. **Add More Resources** → Scenarios become more capable  
3. **Create Smarter Workflows** → Scenarios become more autonomous
4. **Deploy Better Scenarios** → Generate more value and feedback
5. **Improve Core Features** → Cycle continues, getting better each time

## Getting Started

### 1. Identify Your Primary Focus
Read the descriptions above and identify which category best fits your project.

### 2. Use the Appropriate Prompt
Open the corresponding prompt file and follow its comprehensive guidance.

### 3. Consider Dependencies
- If building a scenario, you might need workflows or resources first
- If building workflows, ensure required resources exist
- If building resources, consider how scenarios will use them

### 4. Think About Integration
Every component should enhance the overall Vrooli ecosystem:
- Resources should be usable by multiple scenarios
- Workflows should be reusable and composable  
- Scenarios should demonstrate real business value
- Core features should benefit the entire platform

## Example Decision Process

**"I want to add cryptocurrency wallet capabilities"**
1. This extends Vrooli's capabilities with a new service → **Resource**
2. I'll need workflows to handle transactions → **N8n Workflow** 
3. I'll build a crypto portfolio manager app → **Scenario**

**"I want to improve error handling across the platform"**
1. This affects foundational infrastructure → **Core Features**
2. I might need to update resource error handling → **Resource** (updates)
3. Scenarios will benefit from better error handling → **Scenario** (updates)

**"I want to create an automated content moderation system"**
1. I need workflows for content analysis → **N8n Workflow**
2. I'll build a complete moderation dashboard → **Scenario**
3. I might need new AI resources for content analysis → **Resource**

## Success Principles

Regardless of which prompt you use, follow these principles:

1. **Think Recursively**: How does this make future development easier?
2. **Build for Reuse**: Can other components benefit from your work?
3. **Document Thoroughly**: Make it easy for others (including AI) to understand and extend
4. **Test Comprehensively**: Ensure reliability and integration
5. **Design for Scale**: Consider how this grows with the platform

Remember: You're not just building a feature - you're expanding Vrooli's intelligence and capabilities permanently.