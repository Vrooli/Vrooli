/**
 * Customer Support Swarm Configuration
 * 
 * Demonstrates Tier 1 coordination for a customer support intelligence swarm
 * that dynamically adapts to inquiry complexity and available resources.
 */

import { type SwarmConfig } from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";

export const customerSupportSwarmConfig: SwarmConfig = {
    maxAgents: 10,
    minAgents: 2,
    consensusThreshold: 0.7,
    decisionTimeout: 30000, // 30 seconds
    adaptationInterval: 60000, // 1 minute
    resourceOptimization: true,
    learningEnabled: true,
    
    // Natural language coordination through MCP tools
    coordinationTools: [
        "update_swarm_shared_state",
        "resource_manage",
        "run_routine",
        "send_message",
    ],
    
    // Metacognitive reasoning prompts
    coordinationPrompts: {
        goalDecomposition: "Analyze the customer inquiry and break it down into subtasks based on complexity and required expertise.",
        agentSelection: "Select agents with the appropriate capabilities for each subtask, considering current workload and specialization.",
        conflictResolution: "When agents disagree on approach, facilitate consensus through structured debate and evidence evaluation.",
        performanceReview: "After resolution, analyze the outcome and identify improvements for future similar inquiries.",
    },
};

// Example swarm lifecycle states
export const customerSupportSwarmStates = {
    initialization: {
        id: generatePK(),
        name: "Swarm Initialization",
        description: "Recruiting agents and establishing shared context",
        expectedDuration: "5-10 seconds",
    },
    
    inquiryAnalysis: {
        id: generatePK(),
        name: "Inquiry Analysis",
        description: "Agents collaboratively analyze customer inquiry complexity",
        expectedDuration: "10-20 seconds",
    },
    
    strategyFormation: {
        id: generatePK(),
        name: "Strategy Formation",
        description: "Agents propose and vote on resolution strategies",
        expectedDuration: "15-30 seconds",
    },
    
    execution: {
        id: generatePK(),
        name: "Execution",
        description: "Implementing chosen strategy with coordinated actions",
        expectedDuration: "Variable based on complexity",
    },
    
    review: {
        id: generatePK(),
        name: "Review & Learning",
        description: "Analyzing outcome and updating knowledge base",
        expectedDuration: "10-15 seconds",
    },
};

// Evolution metrics showing improvement over time
export const customerSupportEvolution = {
    week1: {
        avgResolutionTime: "120 seconds",
        avgCost: "$0.45",
        satisfactionRate: "82%",
        firstContactResolution: "65%",
    },
    
    month1: {
        avgResolutionTime: "75 seconds",
        avgCost: "$0.28",
        satisfactionRate: "88%",
        firstContactResolution: "78%",
    },
    
    month3: {
        avgResolutionTime: "45 seconds",
        avgCost: "$0.15",
        satisfactionRate: "94%",
        firstContactResolution: "89%",
    },
    
    improvements: {
        speedGain: "62.5%",
        costReduction: "66.7%",
        satisfactionIncrease: "14.6%",
        fcrIncrease: "36.9%",
    },
};
