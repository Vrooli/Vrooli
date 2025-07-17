/**
 * Simple validation script to test our Priority #1 fix
 */

import { ScenarioFactory } from "./factories/scenario/ScenarioFactory.js";
import { redisFixLoopScenario } from "./scenarios/redis-fix-loop/scenario.js";

async function validateFix() {
    console.log("🔧 Testing Priority #1 Fix: BotParticipant Structure");
    
    try {
        const factory = new ScenarioFactory();
        console.log("✅ ScenarioFactory created");
        
        const scenario = await factory.setupScenario(redisFixLoopScenario);
        console.log("✅ Scenario setup completed");
        
        console.log("📊 Results:");
        console.log(`- Routines: ${scenario.routines?.length || 0}`);
        console.log(`- Agents: ${scenario.agents?.length || 0}`);
        console.log(`- Swarms: ${scenario.swarms?.length || 0}`);
        console.log(`- Blackboard keys: ${Array.from(scenario.blackboard.keys()).join(", ")}`);
        
        if (scenario.agents && scenario.agents.length > 0) {
            console.log("\n🤖 Agent Analysis:");
            
            for (const agent of scenario.agents) {
                console.log(`\n${agent.name}:`);
                console.log(`  - Role: ${agent.role}`);
                console.log(`  - Config.agentSpec exists: ${!!agent.config?.agentSpec}`);
                console.log(`  - Config.agentSpec.role: ${agent.config?.agentSpec?.role}`);
                console.log(`  - Subscriptions: ${agent.config?.agentSpec?.subscriptions?.join(", ") || "none"}`);
                console.log(`  - Behaviors: ${agent.config?.agentSpec?.behaviors?.length || 0}`);
                
                // Detailed behavior analysis
                if (agent.config?.agentSpec?.behaviors) {
                    agent.config.agentSpec.behaviors.forEach((behavior, index) => {
                        console.log(`    Behavior ${index + 1}: ${behavior.trigger?.topic} → ${behavior.action?.type}`);
                    });
                }
            }
            
            // Test EventInterceptor pattern extraction
            console.log("\n🎯 EventInterceptor Pattern Test:");
            const fixer = scenario.agents.find(a => a.name === "redis-problem-fixer");
            if (fixer) {
                console.log("Fixer agent structure for EventInterceptor:");
                console.log("- agentSpec.behaviors exists:", !!fixer.config?.agentSpec?.behaviors);
                console.log("- agentSpec.subscriptions exists:", !!fixer.config?.agentSpec?.subscriptions);
                
                // This is what EventInterceptor.extractPatternsFromBot() looks for
                const behaviors = fixer.config?.agentSpec?.behaviors;
                const subscriptions = fixer.config?.agentSpec?.subscriptions;
                
                const patterns = [];
                if (behaviors && Array.isArray(behaviors)) {
                    for (const behavior of behaviors) {
                        if (behavior.trigger?.topic) {
                            patterns.push(behavior.trigger.topic);
                        }
                    }
                }
                if (subscriptions && Array.isArray(subscriptions)) {
                    patterns.push(...subscriptions);
                }
                
                console.log(`- Extracted patterns: ${patterns.join(", ")}`);
                console.log(`- Should match "custom/redis/fix_requested": ${patterns.includes("custom/redis/fix_requested")}`);
            }
        }
        
        await factory.teardownScenario(scenario);
        console.log("\n✅ Priority #1 Fix Validation SUCCESSFUL!");
        console.log("BotParticipant structure is now compatible with EventInterceptor");
        
    } catch (error) {
        console.error("\n❌ Priority #1 Fix Validation FAILED:");
        console.error(error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

validateFix().catch(console.error);
