/**
 * Customer Inquiry Resolution Routine v1 - Conversational Stage
 * 
 * Initial version using full conversational reasoning for every inquiry.
 * This represents the starting point before pattern recognition and optimization.
 */

import { type RoutineFixture } from "../../../by-domain/types.js";
import { generatePK } from "@vrooli/shared";

export const customerInquiryV1: RoutineFixture = {
    id: generatePK(),
    name: "Customer Inquiry Resolution v1",
    version: "1.0.0",
    description: "Full conversational reasoning for customer inquiries",
    
    config: {
        executionStrategy: "conversational",
        resourceSubType: "CustomerSupport",
        estimatedDuration: "45-60 seconds",
        estimatedCost: "$0.12-0.15 per inquiry",
        
        // Conversational strategy characteristics
        reasoning: {
            approach: "Full chain-of-thought for every inquiry",
            contextWindow: "Entire conversation history",
            toolUsage: "Dynamic tool selection based on reasoning"
        }
    },
    
    // Example execution flow
    executionSteps: [
        {
            name: "Understand Inquiry",
            prompt: "Analyze the customer's message to understand their intent, emotional state, and urgency level.",
            expectedTime: "10-15s"
        },
        {
            name: "Gather Context",
            prompt: "Review customer history, previous interactions, and account details to build complete context.",
            expectedTime: "10-15s"
        },
        {
            name: "Formulate Response",
            prompt: "Generate a personalized response addressing all aspects of the inquiry with empathy and clarity.",
            expectedTime: "15-20s"
        },
        {
            name: "Verify Solution",
            prompt: "Double-check the response for accuracy, completeness, and alignment with company policies.",
            expectedTime: "10s"
        }
    ],
    
    // Performance metrics from production
    metrics: {
        avgExecutionTime: "52 seconds",
        avgCost: "$0.14",
        accuracy: "92%",
        customerSatisfaction: "85%",
        scalabilityIssues: "High latency under load"
    },
    
    // Identified optimization opportunities
    optimizationOpportunities: [
        "60% of inquiries follow predictable patterns",
        "Context gathering often retrieves unused data",
        "Verification step rarely changes the response",
        "Common inquiries could use templates"
    ]
};