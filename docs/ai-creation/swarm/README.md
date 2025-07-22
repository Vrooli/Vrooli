# AI Swarm Creation System

This directory contains the AI-powered swarm creation pipeline for Vrooli. It enables systematic generation, staging, and validation of swarm configurations that serve as the core delivery mechanism across inference-farm, SaaS, and appliance business models.

## Overview

The AI swarm creation system provides **team-based swarm orchestration** through minimal, prompt-driven configurations. Key principles:

- **Team as Template**: Teams serve as minimal swarm configuration templates
- **Chat as Instance**: Chats become running swarm instances spawned from team templates
- **Prompt-Driven Behavior**: Business logic defined through AI prompts, not rigid schemas
- **Emergent Intelligence**: Teams learn and adapt through blackboard sharing and runtime stats
- **Algorithmic Resource Allocation**: Smart resource profiles instead of fixed quotas
- **Economic Emergence**: Profit optimization through AI decision-making, not predefined rules

The system provides:
- **Minimal Configuration**: Core identity, goals, and resource profiles
- **Runtime Intelligence**: Teams learn market patterns and share insights via blackboard
- **Adaptive Behavior**: Business prompts guide economic decisions and resource allocation
- **Performance Tracking**: Lightweight stats that enable algorithmic optimization
- **Future-Proof Design**: New business models = new prompts, not new schemas

### Architecture Model

```
Team (Minimal Template)
├── Core Identity (tier, business model, goal)
├── Business Prompt (economic behavior guidance)
├── Resource Profile (algorithmic allocation)
├── Runtime Stats (performance tracking)
├── Blackboard (shared team intelligence)
└── Spawned Chat Instances
    ├── Chat Instance 1 (Client Project A)
    ├── Chat Instance 2 (Client Project B)
    └── Chat Instance N (Client Project N)
```

## Directory Structure

```
docs/ai-creation/swarm/
├── README.md                    # This file
├── prompts/                     # Swarm generation prompts
│   ├── swarm-generation-prompt.md   # Main generation prompt template
│   ├── business-model-prompt.md     # Business model specific guidance
│   └── vertical-package-prompt.md   # Industry vertical customization
├── backlog.md                   # Queue of swarm ideas to process
├── swarm-reference.json         # Complete reference for all swarms (JSON format)
├── staged/                      # Generated swarm definitions ready for deployment
│   ├── director/                # Meta-coordination swarms
│   ├── infrastructure/          # Platform optimization swarms (I1-I4)
│   ├── profit/                  # Revenue-generating swarms (D/B series)
│   └── appliance/               # Edge deployment swarms
├── schemas/                     # Configuration schemas
│   ├── swarm-config-schema.json    # Extended swarm configuration schema
│   ├── business-model-schema.json  # Business model specific fields
│   └── vertical-package-schema.json # Industry vertical configurations
├── cache/                       # Resolution system cache
│   ├── swarm-index.json         # Available swarms for coordination
│   ├── resource-quotas.json     # Hardware resource allocation patterns
│   └── kpi-benchmarks.json      # Performance benchmarks by swarm type
└── templates/                   # Common swarm patterns
    ├── README.md                # Template usage guide
    ├── director/                # Meta-coordination templates
    ├── infrastructure/          # Platform optimization templates
    ├── profit/                  # Revenue generation templates
    └── appliance/               # Edge deployment templates
```

## Quick Start

### 1. Add Team Swarm Ideas
Edit `backlog.md` to add new team-based swarm concepts:

```markdown
### Team Name
- **Deployment Type**: development|saas|appliance
- **Goal**: Primary business objective and service offering
- **Business Behavior**: How the team should behave economically and operationally
- **Resource Quota**: Specific GPU/RAM/CPU/storage allocation
- **Target Profit**: Monthly profit target in USD
- **Cost Limit**: Maximum monthly cost boundary in USD (optional)
- **Vertical Package**: Industry vertical specialization (optional)
- **Priority**: High|Medium|Low
- **Notes**: Additional context or requirements
```

### 2. Generate Team Configurations

#### Option A: Direct Generation with Claude (Recommended)
```bash
# Generate prompt for manual team configuration creation
./scripts/ai-creation/team-swarm-generate.sh --direct

# This generates a business-model-aware prompt for team configuration
```

#### Option B: Enhanced Generation with Prompt Optimization
```bash
./scripts/ai-creation/team-swarm-generate-enhanced.sh
```

This enhanced system provides:
- **Business Prompt Generation**: AI-crafted prompts for economic behavior
- **Resource Profile Selection**: Smart resource allocation patterns
- **Template Pattern Matching**: Reuse of proven team patterns
- **Emergent Behavior Design**: Prompts that enable adaptive decision-making

### 3. Create and Deploy Teams
Create team configurations with minimal, adaptive setup:

```bash
# Ensure local development environment is running
./scripts/main/develop.sh --target docker --detached yes

# Create team from generated configuration
vrooli team create --config ./docs/ai-creation/swarm/staged/profit/upwork-web-dev-team.json

# Spawn chat instance from team template
vrooli chat create --from-team "upwork-web-development-team" --client-context '{"project": "ecommerce-rebuild"}'

# Monitor team performance and shared learnings
vrooli team monitor --team-id "upwork-web-development-team" --show-blackboard --show-stats
```

### 4. Validate Team Configuration

```bash
# Validate team configuration structure and prompts
./scripts/ai-creation/validate-team-swarm.sh staged/profit/upwork-web-dev-team.json

# Check business prompt effectiveness
./scripts/ai-creation/validate-team-prompts.sh

# Verify resource profile alignment
./scripts/ai-creation/validate-team-resources.sh
```

## Deployment Types and Service Categories

### Deployment Type Matrix

| Service Category | Development | SaaS | Appliance |
|------------------|-------------|------|-----------|
| **Coordination** | Fleet economic optimization, swarm spawning/termination | User growth optimization, feature prioritization | Edge orchestration, customer success automation |
| **Infrastructure** | Platform optimization, research, content generation | Feature development, UX optimization, marketing | Edge updates, compliance monitoring, support |
| **Revenue Services** | Upwork teams, consulting services, specialized tools | Subscription features, premium tools, enterprise solutions | Industry workflows, vertical automations, value-add services |
| **Edge Processing** | N/A (server-based) | N/A (cloud-based) | Edge processing, local AI, customer-premises deployment |

### Service Category Definitions

#### 🎯 **Coordination Services**
- **Purpose**: Meta-coordination and business optimization
- **Behavior**: AI-driven decision making for spawning/termination, resource allocation
- **Intelligence**: Learn from all team performance data, optimize fleet economics
- **Resource Profile**: Heavy (requires significant computational power for coordination)

#### 🏗️ **Infrastructure Services** 
- **I₁ Platform**: Core Vrooli platform optimization and reliability
- **I₂ UX**: User experience enhancement and interface optimization  
- **I₃ Research**: Experimental workflows and capability development
- **I₄ Marketing**: Content generation, lead acquisition, brand building
- **Behavior**: Focus on long-term platform value over immediate profit
- **Intelligence**: Share optimization insights across all infrastructure teams

#### 💰 **Revenue Services**
- **D-Series**: High-value consulting and custom development services
- **B-Series**: Scalable products and recurring revenue streams
- **Behavior**: Aggressive profit optimization while maintaining quality
- **Intelligence**: Share market insights, pricing strategies, client patterns
- **Resource Profile**: Typically standard, scales to heavy based on demand

#### 📱 **Edge Processing Services**
- **Purpose**: Customer-premises deployment for SMBs
- **Behavior**: Autonomous customer value delivery with minimal human intervention
- **Intelligence**: Learn customer patterns, optimize vertical workflows
- **Resource Profile**: Light to standard (edge constraints with cloud spillover)

## Team Configuration Schema (Source of Truth: packages/shared/src/shape/configs/team.ts)

### Core Team Configuration
```typescript
interface TeamConfigObject extends BaseConfigObject {
  // Core identity (minimal)
  deploymentType: "development" | "saas" | "appliance";
  goal: string;
  
  // Behavior definition (prompt-driven like bots)
  businessPrompt: string;
  
  // Resource allocation
  resourceQuota: {
    gpuPercentage: number;  // 0-100
    ramGB: number;
    cpuCores: number;
    storageGB: number;
  };
  
  // Economic targets (in micro-dollars as stringified bigints)
  targetProfitPerMonth: string;  // e.g. "2800000000" for $2800
  costLimit?: string;            // Optional safety boundary
  
  // Industry specialization (optional)
  verticalPackage?: {
    industry: string;
    complianceRequirements?: string[];
    defaultWorkflows?: string[];
    dataPrivacyLevel: "standard" | "hipaa" | "pci" | "sox" | "gdpr";
    terminology?: Record<string, string>;
    regulatoryBodies?: string[];
  };
  
  // Market targeting (optional)
  marketSegment?: "enterprise" | "smb" | "consumer";
  
  // Security and isolation (optional)
  isolation?: {
    sandboxLevel: "none" | "user-namespace" | "full-container" | "vm";
    networkPolicy: "open" | "restricted" | "isolated" | "air-gapped";
    secretsAccess: string[];
    auditLogging?: boolean;
    securityPolicies?: string[];
  };
  
  // AI model preference (optional)
  preferredModel?: string;
  
  // Runtime intelligence (like chat stats)
  stats: {
    totalInstances: number;
    totalProfit: string;           // stringified bigint
    totalCosts: string;            // stringified bigint
    averageKPIs: Record<string, number>;
    activeInstances: number;
    lastUpdated: number;
  };
  
  // Team blackboard for shared learning (optional)
  blackboard?: BlackboardItem[];
  
  // Spawning defaults (optional)
  defaultLimits?: {
    maxToolCallsPerBotResponse: number;
    maxToolCalls: number;
    maxCreditsPerBotResponse: string;
    maxCredits: string;
    maxDurationPerBotResponseMs: number;
    maxDurationMs: number;
    delayBetweenProcessingCyclesMs: number;
  };
  defaultScheduling?: {
    defaultDelayMs: number;
    toolSpecificDelays: Record<string, number>;
    requiresApprovalTools: string[] | "all" | "none";
    approvalTimeoutMs: number;
    autoRejectOnTimeout: boolean;
  };
  defaultPolicy?: {
    visibility: "open" | "restricted" | "private";
    acl?: string[];
  };
}
```

## Swarm Templates

### Fleet Coordinator Template
```json
{
  "__version": "1.0",
  "deploymentType": "development",
  "goal": "Optimize fleet economics and coordinate swarm lifecycle for maximum profitability",
  "businessPrompt": "You are the Director of the Vrooli inference farm. Your job is to maximize fleet profitability by intelligently spawning and terminating profit swarms based on market conditions, resource utilization, and performance data. Monitor all team blackboards for insights. Make data-driven decisions about resource allocation. Target 50% reinvestment rate when monthly profit exceeds $4000. Share fleet optimization insights via your blackboard.",
  "resourceQuota": {
    "gpuPercentage": 35,
    "ramGB": 32,
    "cpuCores": 8,
    "storageGB": 200
  },
  "targetProfitPerMonth": "10000000000",
  "costLimit": "2000000000",
  "stats": {
    "totalInstances": 0,
    "totalProfit": "0",
    "totalCosts": "0",
    "averageKPIs": {},
    "activeInstances": 0,
    "lastUpdated": 0
  },
  "blackboard": []
}
```

### Revenue Service Template (Upwork Team)
```json
{
  "__version": "1.0",
  "deploymentType": "development",
  "goal": "Deliver high-quality web development projects on Upwork with 4.7+ client satisfaction",
  "businessPrompt": "You are an Upwork web development team focused on profitable project delivery. Target $2500+/month profit across all instances. Maintain 75%+ profit margins. Prioritize projects $800+ value. Monitor GPU utilization - scale instance count based on project queue. Share learnings via team blackboard: successful project patterns, client feedback trends, optimal pricing strategies. Coordinate with director for spawning decisions when queue exceeds capacity.",
  "resourceQuota": {
    "gpuPercentage": 20,
    "ramGB": 16,
    "cpuCores": 4,
    "storageGB": 100
  },
  "targetProfitPerMonth": "2500000000",
  "costLimit": "800000000",
  "stats": {
    "totalInstances": 0,
    "totalProfit": "0",
    "totalCosts": "0",
    "averageKPIs": {},
    "activeInstances": 0,
    "lastUpdated": 0
  },
  "blackboard": []
}
```

### Edge Service Template (Dental Office)
```json
{
  "__version": "1.0",
  "deploymentType": "appliance",
  "goal": "Automate dental office admin tasks with HIPAA compliance and 4.7+ patient satisfaction",
  "businessPrompt": "You are a dental office automation assistant deployed on customer premises. Prioritize patient satisfaction and HIPAA compliance above all else. Handle appointment scheduling, insurance verification, follow-up reminders, and basic administrative tasks. Learn office patterns and preferences via blackboard. Target $200/month value delivery with <5% churn. Process data locally when possible, use cloud APIs only when necessary. Share anonymized efficiency insights with other dental appliances.",
  "resourceQuota": {
    "gpuPercentage": 10,
    "ramGB": 8,
    "cpuCores": 2,
    "storageGB": 50
  },
  "targetProfitPerMonth": "200000000",
  "costLimit": "60000000",
  "verticalPackage": "dental-practice",
  "stats": {
    "totalInstances": 0,
    "totalProfit": "0",
    "totalCosts": "0",
    "averageKPIs": {},
    "activeInstances": 0,
    "lastUpdated": 0
  },
  "blackboard": []
}
```

## Resource Allocation (Source: packages/shared/src/shape/configs/team.ts)

### Standard Resource Quotas
Teams use direct resource allocation through the `resourceQuota` field:

```typescript
interface ResourceQuota {
  gpuPercentage: number;  // GPU allocation percentage (0-100)
  ramGB: number;          // RAM allocation in gigabytes
  cpuCores: number;       // CPU core allocation
  storageGB: number;      // Storage allocation in gigabytes
}
```

### Predefined Resource Quotas (STANDARD_RESOURCE_QUOTAS)
```typescript
const STANDARD_RESOURCE_QUOTAS = {
  light: { gpuPercentage: 10, ramGB: 8, cpuCores: 2, storageGB: 50 },     // Edge deployment
  standard: { gpuPercentage: 20, ramGB: 16, cpuCores: 4, storageGB: 100 }, // Typical profit swarms
  heavy: { gpuPercentage: 35, ramGB: 32, cpuCores: 8, storageGB: 200 },   // Director swarms
  ultra: { gpuPercentage: 50, ramGB: 64, cpuCores: 16, storageGB: 500 }   // Ultra-heavy workloads
};
```

### Dynamic Scaling
While teams specify initial resource quotas, the system can dynamically adjust based on:
- Instance count and workload patterns
- Performance metrics and success rates
- Market demand and scaling requirements
- Hardware utilization across the fleet

## Emergent Intelligence System

### Team Blackboard Learning
```typescript
interface TeamInsight {
  type: "market-pattern" | "resource-optimization" | "client-feedback" | "pricing-strategy";
  data: unknown;
  confidence: number;
  created_at: string;
}

// Example insights teams discover:
{
  type: "market-pattern",
  data: { "best_project_types": ["e-commerce", "SaaS"], "avoid": ["crypto"] },
  confidence: 0.85
}

{
  type: "resource-optimization", 
  data: { "optimal_gpu_allocation": 15, "peak_hours": [9, 14, 20] },
  confidence: 0.92
}
```

### AI-Driven Decision Making
Teams make economic decisions through business prompts:
- Should spawn new instance? (AI analyzes market conditions + team performance)
- How to price services? (AI learns from successful patterns shared via blackboard)
- When to terminate underperforming instances? (AI evaluates cost/benefit in context)
- How to allocate resources? (Algorithm adapts based on performance data)

## Director Swarm Intelligence

### AI-Driven Fleet Management
Director swarms use business prompts to make decisions:

```typescript
// Director decision-making prompt example
const directorPrompt = `
You are managing a fleet of profit swarms. Current state:
- Fleet profit: $12,500/month
- Resource utilization: 73%
- Upwork queue: 8 projects pending
- Web dev team performance: 92% success rate, $18/agent-hour

Should you spawn a new web dev team instance?
Consider: profit opportunity, resource availability, reinvestment strategy (50% above $4k profit).
`;
```

### Emergent Fleet Optimization
Directors learn through blackboard intelligence:
- Read insights from all profit teams
- Identify patterns across different markets
- Optimize resource allocation algorithmically
- Make spawning/termination decisions based on collective intelligence

No rigid rules - pure AI decision-making guided by business prompts and shared learning.

## Best Practices

### Prompt-Driven Design Principles

#### **Minimal Configuration**
- Core identity, goal, and business prompt define behavior
- Resource profiles replace rigid quotas
- Profit hints guide decisions without constraining adaptation

#### **Emergent Intelligence**
- Teams learn through blackboard sharing
- AI makes economic decisions via business prompts
- Patterns emerge from collective experience, not predefined rules

#### **Adaptive Resource Management**
- Algorithmic allocation based on performance and demand
- Dynamic scaling through AI decision-making
- Resource optimization learned through fleet intelligence

### Emergence Best Practices

#### **Business Prompt Design**
- Clear economic objectives and behavioral guidance
- Specific coordination instructions for blackboard usage
- Adaptive decision-making framework, not rigid rules

#### **Blackboard Intelligence**
- Teams share market insights, not just performance metrics
- Confidence scoring for insight reliability
- Cross-team learning patterns for fleet optimization

#### **Runtime Adaptation**
- Resource allocation adjusts based on performance data
- Economic decisions made through AI analysis, not thresholds
- Continuous learning from instance outcomes and market feedback

## Troubleshooting

### Prompt Issues
- **"Team not behaving economically"**: Refine business prompt with clearer objectives
- **"Poor coordination"**: Update prompt with better blackboard usage instructions
- **"Inconsistent decisions"**: Add more context and guidance to business prompt

### Intelligence Issues
- **"Teams not learning"**: Check blackboard activity and insight generation
- **"Poor performance tracking"**: Verify stats updates and KPI calculation
- **"No cross-team insights"**: Review blackboard sharing patterns and confidence scores

### Resource Issues
- **"Resource allocation inefficient"**: Review profile selection and algorithmic allocation
- **"Performance degradation"**: Analyze instance workload and resource profile matching
- **"Cost overruns"**: Check profit hints vs actual costs, adjust behavioral prompts

## Scripts and Utilities

### Prompt Analysis
```bash
# Validate business prompt effectiveness
./validate-team-prompts.sh staged/profit/upwork-team.json

# Generate optimized business prompts for team types
./generate-business-prompts.sh --tier profit --business-model inference-farm

# Test prompt decision-making consistency
./test-prompt-decisions.sh --team upwork-team --scenarios market-conditions.json
```

### Intelligence Monitoring
```bash
# Monitor team blackboard activity and insights
./monitor-team-intelligence.sh --team upwork-team --show-insights

# Analyze cross-team learning patterns
./analyze-fleet-intelligence.sh --timeframe 30d --confidence-threshold 0.8

# Export team insights for analysis
./export-team-insights.sh --format json --teams all --output fleet-intelligence.json
```

### Resource Optimization
```bash
# Monitor algorithmic resource allocation
./monitor-resource-profiles.sh --show-efficiency --teams all

# Analyze resource usage vs performance correlation
./analyze-resource-performance.sh --timeframe 30d --output resource-analysis.json

# Optimize resource profiles based on performance data
./optimize-resource-profiles.sh --teams underperforming --auto-adjust
```

## Related Documentation

- [`../agent/README.md`](../agent/README.md) - Agent creation for swarm participants
- [`../routine/README.md`](../routine/README.md) - Routine creation for swarm workflows
- [`/docs/architecture/execution/README.md`](../../architecture/execution/README.md) - Swarm execution architecture
- [`/docs/plans/swarm-orchestration.md`](../../plans/swarm-orchestration.md) - Swarm coordination strategies
- [`IMPLEMENTATION_SUMMARY.md`](./IMPLEMENTATION_SUMMARY.md) - Current implementation status and next steps