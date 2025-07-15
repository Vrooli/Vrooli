# AI Agent Import Guide for Vrooli

This guide explains how to import the 124 AI-generated agents from the staged directory into the Vrooli app.

## Overview

The AI agent creation system has generated:
- **59 Specialist agents** - Domain experts and tool specialists
- **26 Monitor agents** - System watchers and quality assurance
- **24 Coordinator agents** - Swarm orchestrators and resource managers  
- **15 Bridge agents** - Human-AI interfaces and platform integrators

Total: **124 agents** ready for import

## Agent Structure

Each agent JSON file contains an `AgentSpec` configuration that defines:
- **Identity**: Name and version
- **Goal**: Primary objective
- **Role**: Function within the swarm (coordinator/specialist/monitor/bridge)
- **Subscriptions**: Event topics the agent listens to
- **Behaviors**: Event-driven actions (routine calls or reasoning invokes)
- **Resources**: Access to routines, blackboard, and other tools

## Import Process

### Prerequisites

1. **Local development environment running**:
   ```bash
   ./scripts/main/develop.sh --target docker --detached yes
   ```

2. **Database with user/bot support**:
   - Ensure the database schema supports `botSettings` on User model
   - Bot configuration stored as JSON in `botSettings` field

### Import Methods

#### Method 1: Direct Database Import (Recommended for Bulk)

Since agents are stored as bot configurations on User records, you can:

1. **Create bot users programmatically**:
   ```typescript
   // For each agent JSON file:
   const agentSpec = JSON.parse(fs.readFileSync(agentFile));
   
   // Create a bot user with the agent spec
   const bot = await prisma.user.create({
     data: {
       handle: agentSpec.identity.name,
       name: agentSpec.identity.name,
       isBot: true,
       botSettings: {
         __version: "1.0",
         resources: agentSpec.resources || [],
         agentSpec: agentSpec
       }
     }
   });
   ```

2. **Batch import script**:
   ```bash
   # Create a script to import all agents
   cat > import-agents.ts << 'EOF'
   import { PrismaClient } from '@prisma/client';
   import fs from 'fs';
   import path from 'path';

   const prisma = new PrismaClient();

   async function importAgents() {
     const agentDirs = ['coordinator', 'specialist', 'monitor', 'bridge'];
     
     for (const dir of agentDirs) {
       const dirPath = path.join('docs/ai-creation/agent/staged', dir);
       const files = fs.readdirSync(dirPath).filter(f => f.endsWith('.json'));
       
       for (const file of files) {
         const agentData = JSON.parse(fs.readFileSync(path.join(dirPath, file), 'utf8'));
         
         // Check if bot already exists
         const existing = await prisma.user.findUnique({
           where: { handle: agentData.identity.name }
         });
         
         if (existing) {
           console.log(`Updating agent: ${agentData.identity.name}`);
           await prisma.user.update({
             where: { id: existing.id },
             data: {
               botSettings: {
                 __version: "1.0",
                 resources: agentData.resources || [],
                 agentSpec: agentData
               }
             }
           });
         } else {
           console.log(`Creating agent: ${agentData.identity.name}`);
           await prisma.user.create({
             data: {
               handle: agentData.identity.name,
               name: agentData.identity.name,
               isBot: true,
               isPrivate: false,
               botSettings: {
                 __version: "1.0",
                 resources: agentData.resources || [],
                 agentSpec: agentData
               }
             }
           });
         }
       }
     }
   }

   importAgents()
     .then(() => console.log('All agents imported successfully'))
     .catch(console.error)
     .finally(() => prisma.$disconnect());
   EOF
   ```

#### Method 2: API-Based Import

If there's an API endpoint for creating/updating bot users:

```bash
# For each agent file
for agent in docs/ai-creation/agent/staged/*/*.json; do
  agentData=$(cat "$agent")
  agentName=$(echo "$agentData" | jq -r '.identity.name')
  
  # Create bot user via API
  curl -X POST http://localhost:5329/api/user/create \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -d "{
      \"handle\": \"$agentName\",
      \"name\": \"$agentName\",
      \"isBot\": true,
      \"botSettings\": {
        \"__version\": \"1.0\",
        \"resources\": $(echo "$agentData" | jq '.resources // []'),
        \"agentSpec\": $agentData
      }
    }"
done
```

#### Method 3: CLI Extension (Future)

Extend the Vrooli CLI to support agent imports:

```bash
# Add to packages/cli/src/commands/agent.ts
vrooli agent import ./docs/ai-creation/agent/staged/coordinator/task-orchestration-coordinator.json
vrooli agent import-dir ./docs/ai-creation/agent/staged/
```

## Validation Before Import

1. **Check routine dependencies**:
   ```bash
   # Ensure all referenced routines exist
   ./docs/ai-creation/agent/validate-agent-behaviors.sh
   ```

2. **Verify event subscriptions**:
   - All subscriptions must use predefined platform events
   - Valid events: swarm/*, run/*, step/*, safety/*

3. **Test agent configurations**:
   - Import a few agents first to test the process
   - Verify they appear in the bot user list
   - Check botSettings are properly stored

## Post-Import Steps

1. **Verify agents in database**:
   ```sql
   SELECT handle, name, "botSettings" 
   FROM users 
   WHERE "isBot" = true 
   AND "botSettings" IS NOT NULL
   ORDER BY "createdAt" DESC;
   ```

2. **Test agent activation**:
   - Create a swarm with relevant agents
   - Trigger events that match agent subscriptions
   - Monitor agent behaviors and responses

3. **Generate agent reference**:
   ```bash
   # Create/update agent-reference.json with imported agents
   # This helps track which agents are available in the system
   ```

## Troubleshooting

### Common Issues

1. **"Routine not found" errors**:
   - Ensure all routines referenced in agent behaviors are imported first
   - Use routine-reference.json to map routine names to IDs

2. **"Invalid subscription" errors**:
   - Check that all topics match predefined platform events
   - No custom event topics are allowed

3. **"Bot creation failed"**:
   - Verify botSettings JSON structure matches BotConfig interface
   - Ensure handle is unique (no duplicate agent names)

### Debug Commands

```bash
# List all staged agents
find docs/ai-creation/agent/staged -name "*.json" | sort

# Validate specific agent
jq . docs/ai-creation/agent/staged/coordinator/task-orchestration-coordinator.json

# Check for TODO placeholders
grep -r "TODO" docs/ai-creation/agent/staged/

# Count agents by category
for dir in docs/ai-creation/agent/staged/*/; do
  echo "$(basename $dir): $(find $dir -name "*.json" | wc -l)"
done
```

## Next Steps

1. **Bulk Import**: Run the import script to load all 124 agents
2. **Testing**: Create test swarms to validate agent behaviors
3. **Monitoring**: Set up logging to track agent performance
4. **Optimization**: Refine agent configurations based on real usage

## Related Documentation

- [Agent README](./README.md) - Complete agent creation system docs
- [Bot Configuration](../../../packages/shared/src/shape/configs/bot.ts) - AgentSpec interface
- [Routine Import Guide](../routine/README.md) - Import routines that agents use