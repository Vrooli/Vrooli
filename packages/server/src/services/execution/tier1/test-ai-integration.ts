/**
 * Manual test script to verify AI integration in Tier 1
 * 
 * Run with: npx tsx test-ai-integration.ts
 */

import winston from "winston";
import { EventBus } from "../cross-cutting/events/eventBus.js";
import { InMemorySwarmStateStore } from "./state/swarmStateStore.js";
import { SwarmStateMachine } from "./coordination/swarmStateMachine.js";
import { StrategyEngine } from "./intelligence/strategyEngine.js";
import { ConversationBridge } from "./intelligence/conversationBridge.js";
import { SwarmState as SwarmStateEnum } from "@vrooli/shared";

const logger = winston.createLogger({
    level: "info",
    format: winston.format.simple(),
    transports: [new winston.transports.Console()],
});

async function testAIIntegration() {
    console.log("🧪 Testing AI Integration in Tier 1...\n");

    // Test 1: ConversationBridge basic reasoning
    console.log("1️⃣ Testing ConversationBridge reasoning...");
    const bridge = new ConversationBridge(logger);
    
    try {
        const response = await bridge.reason(
            "What are the key steps to deploy a web application?",
            { context: "deployment planning" }
        );
        console.log("✅ AI Response:", response.substring(0, 200) + "...\n");
    } catch (error) {
        console.error("❌ ConversationBridge test failed:", error);
    }

    // Test 2: StrategyEngine analysis
    console.log("2️⃣ Testing StrategyEngine situation analysis...");
    const eventBus = new EventBus(logger);
    const strategyEngine = new StrategyEngine(logger, eventBus);
    
    try {
        const analysis = await strategyEngine.analyzeSituation({
            goal: "Build a task management application",
            observations: {
                currentPhase: "planning",
                teamSize: 0,
                resourcesAvailable: true,
            },
            knowledge: {
                facts: new Map(),
                insights: [],
                decisions: [],
                swarmId: "test-swarm",
            },
            progress: {
                tasksCompleted: 0,
                tasksTotal: 0,
                milestones: [],
                currentPhase: "initialization",
            },
        });
        console.log("✅ Strategic Analysis:", analysis.substring(0, 200) + "...\n");
    } catch (error) {
        console.error("❌ StrategyEngine test failed:", error);
    }

    // Test 3: Decision generation
    console.log("3️⃣ Testing decision generation...");
    try {
        const decisions = await strategyEngine.generateDecisions({
            goal: "Build a task management application",
            orientation: {
                situation: "Starting new project",
                opportunities: ["Use existing UI components", "Leverage task APIs"],
                threats: ["Limited time budget"],
            },
            constraints: {
                budget: 1000,
                timeLimit: 7200000, // 2 hours
            },
        });
        console.log("✅ Generated Decisions:", decisions);
    } catch (error) {
        console.error("❌ Decision generation test failed:", error);
    }

    console.log("\n🎉 AI Integration test complete!");
    process.exit(0);
}

// Run the test
testAIIntegration().catch((error) => {
    console.error("Test failed with error:", error);
    process.exit(1);
});