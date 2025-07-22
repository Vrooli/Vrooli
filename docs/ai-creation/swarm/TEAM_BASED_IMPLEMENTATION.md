# Team-Based Swarm Implementation Guide

This guide explains how the team-based swarm architecture enables scalable, economically optimized swarm deployment for Vrooli's business models.

## Architecture Overview

### Core Concept: Teams as Swarm Templates

```
┌─────────────────────────────────────────────────────────────┐
│                      TEAM CONFIGURATION                     │
│                    (Swarm Template)                         │
├─────────────────────────────────────────────────────────────┤
│ • Business Model: inference-farm, saas, appliance         │
│ • Economic Targets: $2400/month profit, 75% margin        │
│ • Resource Quotas: 20% GPU, 24GB RAM, full isolation      │
│ • KPI Thresholds: 92% success, $18/agent-hour             │
│ • Vertical Package: consulting, dental, saas-tools        │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ spawns multiple instances
                              ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  CHAT INSTANCE  │  │  CHAT INSTANCE  │  │  CHAT INSTANCE  │
│   (Client A)    │  │   (Client B)    │  │   (Client C)    │
├─────────────────┤  ├─────────────────┤  ├─────────────────┤
│ • Project: Web  │  │ • Project: API  │  │ • Project: CRM  │
│ • Budget: $800  │  │ • Budget: $1200 │  │ • Budget: $2000 │
│ • Duration: 2w  │  │ • Duration: 3w  │  │ • Duration: 4w  │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Benefits of Team-Based Architecture

1. **Economic Aggregation**: Track profitability across all instances of a service type
2. **Template Reuse**: One configuration spawns many optimized instances
3. **Director Control**: Meta-swarms manage spawning/termination at team level
4. **Business Alignment**: Teams = service offerings, instances = client engagements
5. **Resource Optimization**: Shared quotas with performance-based adjustments

## Implementation Details

### Team Configuration Schema

Teams use the `TeamConfigObject` in `packages/shared/src/shape/configs/team.ts`:

```typescript
interface TeamConfigObject extends BaseConfigObject {
  // Core identity
  deploymentType: "development" | "saas" | "appliance";
  goal: string;
  businessPrompt: string;
  
  // Resource allocation
  resourceQuota: {
    gpuPercentage: number;    // 0-100% of available GPU
    ramGB: number;            // RAM allocation in gigabytes
    cpuCores: number;         // CPU core allocation
    storageGB: number;        // Storage allocation in gigabytes
  };
  
  // Economic configuration (in micro-dollars as stringified bigints)
  targetProfitPerMonth: string;  // e.g. "2800000000" for $2800
  costLimit?: string;            // Maximum monthly cost boundary
  
  // Optional configurations
  verticalPackage?: string;      // Industry vertical specialization
  marketSegment?: "enterprise" | "smb" | "consumer";
  preferredModel?: string;       // AI model preference
  
  // Runtime intelligence
  stats: TeamEconomicTracking;   // Performance and economic tracking
  blackboard?: BlackboardItem[]; // Shared team intelligence
  
  // Instance defaults
  defaultLimits?: ChatLimits;
  defaultScheduling?: ChatScheduling;
  defaultPolicy?: ChatPolicy;
}
```

### Chat Instance Creation

When creating a chat from a team template:

```typescript
// Enhanced ChatConfigObject reference to team
interface ChatConfigObject {
  // Existing chat fields...
  teamTemplate: {
    teamId: string;              // Reference to team configuration
    templateVersion: string;     // Team config version for updates
    instanceOverrides?: {        // Instance-specific customizations
      maxCredits?: string;
      clientContext?: any;
    };
    clientContext?: {            // Client-specific data
      projectName: string;
      budget: number;
      requirements: string[];
    };
  };
}
```

## Business Model Implementation

### Inference Farm Model

**Teams as Service Offerings**:
- "Upwork Web Development Team" (D-series profit swarm)
- "Schema Builder SaaS Team" (B-series profit swarm)  
- "Platform Optimizer I1" (infrastructure swarm)
- "Fleet Economic Director" (director swarm)

**Implementation Pattern**:
```typescript
// Director swarm monitors team performance
const teamPerformance = await getTeamEconomicTracking(teamId);
if (teamPerformance.profitPerInstance > profitThreshold) {
  await spawnChatFromTeam(teamId, demandSignal);
}
```

### SaaS Model

**Teams as Feature Areas**:
- "User Onboarding Optimization" (infrastructure I2)
- "Feature Development Pipeline" (infrastructure I1)
- "Customer Success Automation" (infrastructure I4)
- "Analytics & Insights Engine" (infrastructure I3)

**Implementation Pattern**:
```typescript
// Feature teams optimize user experience
const userMetrics = await getUserExperienceMetrics();
if (userMetrics.churnRate > churnThreshold) {
  await spawnChatFromTeam("customer-success-automation", userSegment);
}
```

### Appliance Model

**Teams as Vertical Packages**:
- "Dental Office Assistant" (appliance tier)
- "Landscaping Business Optimizer" (appliance tier)
- "Nursery Management System" (appliance tier)
- "Legal Practice Automation" (appliance tier)

**Implementation Pattern**:
```typescript
// Each appliance deployment gets chat instance
const applianceSetup = await detectApplianceSetup();
if (applianceSetup.verticalPackage === "dental") {
  await spawnChatFromTeam("dental-office-assistant", officeConfig);
}
```

## Director Swarm Integration

### Team-Level Management

Director swarms operate at the team level rather than individual chat level:

```typescript
class DirectorSwarmLogic {
  async optimizeFleetEconomics() {
    const teams = await getAllProfitTeams();
    
    for (const team of teams) {
      const tracking = team.economicTracking;
      
      // Spawn new instances for profitable teams
      if (team.shouldSpawnNewInstance()) {
        await this.spawnChatFromTeam(team.id);
      }
      
      // Terminate underperforming instances
      if (team.shouldTerminateInstances()) {
        await this.terminateWorstPerformingChats(team.id);
      }
      
      // Adjust resource allocation based on performance
      const newQuotas = team.getEffectiveResourceAllocation();
      await this.updateTeamResourceQuotas(team.id, newQuotas);
    }
  }
}
```

### Economic Decision Making

```typescript
// Team-level economic signals
interface TeamEconomicSignals {
  profitPerInstance: number;
  instanceSpawnRate: number;
  averageClientSatisfaction: number;
  resourceUtilizationRate: number;
  competitorPricingTrends: number;
}

// Director decisions based on team economics
function makeSpawningDecision(team: TeamConfig, signals: TeamEconomicSignals): SpawningDecision {
  if (signals.profitPerInstance > team.economics.targetProfitPerMonth * 0.8) {
    return { action: "spawn", priority: "high" };
  }
  
  if (signals.resourceUtilizationRate > 90) {
    return { action: "scale-resources", priority: "medium" };
  }
  
  if (signals.averageClientSatisfaction < 4.0) {
    return { action: "optimize-quality", priority: "high" };
  }
  
  return { action: "monitor", priority: "low" };
}
```

## Implementation Examples

### Creating an Upwork Web Development Team

```typescript
// 1. Create team configuration
const upworkTeamConfig: TeamConfigObject = {
  __version: "1.0",
  businessModel: {
    type: "inference-farm",
    tier: "profit",
    subTier: "D-series",
    marketSegment: "smb"
  },
  economics: {
    targetProfitPerMonth: 2800,
    maxMonthlyCost: 600,
    breakEvenThreshold: 1500,
    revenueModel: "upwork",
    marginTarget: 75
  },
  isolation: {
    sandboxLevel: "full-container",
    resourceQuotas: {
      gpuQuota: 20,
      ramQuota: 24,
      cpuQuota: 6
    },
    networkIsolation: true,
    secretsVault: "/vault/profit/upwork"
  },
  kpis: {
    primary: {
      profitPerAgentHour: 18,
      successRate: 92
    },
    customer: {
      customerSatisfaction: 4.7
    },
    thresholds: {
      criticalProfit: 800,
      warningProfit: 1200
    }
  },
  swarmGoal: "Deliver high-quality web development projects with 92%+ success rate and $2800+ monthly profit",
  economicTracking: defaultEconomicTracking()
};

// 2. Create team in database
const team = await createTeam({
  name: "Upwork Web Development Team",
  config: JSON.stringify(upworkTeamConfig)
});

// 3. Director spawns instances based on demand
const demandSignal = await detectUpworkDemand("web-development");
if (demandSignal.projectsAvailable > 2) {
  const chatInstance = await createChatFromTeam(team.id, {
    clientContext: {
      projectType: "ecommerce-rebuild",
      budget: 800,
      timeline: "2 weeks"
    }
  });
}
```

### Spawning Chat Instances

```typescript
async function createChatFromTeam(teamId: string, options?: {
  clientContext?: any;
  instanceOverrides?: Partial<TeamConfigObject>;
}): Promise<Chat> {
  
  const team = await getTeamWithConfig(teamId);
  const teamConfig = TeamConfig.parse(team, logger);
  
  // Create chat with team template reference
  const chatConfig: ChatConfigObject = {
    ...ChatConfig.default().export(),
    
    // Reference to team template
    teamTemplate: {
      teamId: team.id,
      templateVersion: teamConfig.__version,
      instanceOverrides: options?.instanceOverrides,
      clientContext: options?.clientContext
    },
    
    // Apply team defaults
    goal: teamConfig.swarmGoal,
    limits: teamConfig.defaultLimits,
    scheduling: teamConfig.defaultScheduling,
    policy: teamConfig.defaultPolicy,
    
    // Team-specific resource allocation
    resourceQuotas: teamConfig.getEffectiveResourceAllocation()
  };
  
  const chat = await createChat({
    config: JSON.stringify(chatConfig)
  });
  
  // Update team economic tracking
  teamConfig.updateEconomicTracking({
    profit: 0, // Will be updated as instance runs
    costs: 0,
    kpis: teamConfig.kpis.primary,
    isNewInstance: true
  });
  
  await updateTeam(teamId, {
    config: JSON.stringify(teamConfig.export())
  });
  
  return chat;
}
```

## Migration Strategy

### Phase 1: Enhance Team Entity
1. Update `TeamConfigObject` schema (✅ Complete)
2. Add economic tracking fields
3. Implement team-level KPI aggregation
4. Create team configuration validation

### Phase 2: Update Chat Creation
1. Add `teamTemplate` field to `ChatConfigObject`
2. Implement `createChatFromTeam()` function
3. Add instance-level economic tracking
4. Update chat termination to update team metrics

### Phase 3: Director Integration
1. Update director swarms to monitor teams instead of individual chats
2. Implement team-level spawning/termination logic
3. Add resource allocation optimization based on team performance
4. Create economic aggregation and reporting

### Phase 4: Business Model Alignment
1. Create team templates for each business model
2. Implement vertical package configurations
3. Add compliance and security validation
4. Deploy initial profitable team configurations

## Economic Tracking Flow

```typescript
// Instance lifecycle economic tracking
class TeamEconomicTracker {
  
  async onChatStarted(chatId: string, teamId: string) {
    const team = await getTeamConfig(teamId);
    team.updateEconomicTracking({
      profit: 0,
      costs: 0,
      kpis: team.kpis.primary,
      isNewInstance: true
    });
    await saveTeamConfig(teamId, team);
  }
  
  async onChatCompleted(chatId: string, teamId: string, results: ChatResults) {
    const team = await getTeamConfig(teamId);
    team.updateEconomicTracking({
      profit: results.revenue - results.costs,
      costs: results.costs,
      kpis: {
        profitPerAgentHour: results.profitPerHour,
        successRate: results.clientSatisfaction >= 4.0 ? 100 : 0,
        customerSatisfaction: results.clientSatisfaction
      },
      isInstanceClosed: true
    });
    await saveTeamConfig(teamId, team);
    
    // Signal director swarm about performance
    await publishEvent("team/instance/completed", {
      teamId,
      performance: team.economicTracking
    });
  }
}
```

This team-based architecture provides the foundation for your logarithmic profit scaling model, enabling director swarms to make intelligent spawning/termination decisions based on aggregated team performance rather than individual chat metrics.