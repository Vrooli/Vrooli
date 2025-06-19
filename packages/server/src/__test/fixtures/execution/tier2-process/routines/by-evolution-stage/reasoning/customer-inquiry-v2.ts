/**
 * Customer Inquiry Resolution Routine v2 - Reasoning Stage
 * 
 * Evolved version using pattern recognition and structured reasoning frameworks.
 * Agents identified common patterns and created optimized reasoning paths.
 */

import { type RoutineFixture } from "../../../by-domain/types.js";
import { generatePK } from "@vrooli/shared";

export const customerInquiryV2: RoutineFixture = {
    id: generatePK(),
    name: "Customer Inquiry Resolution v2",
    version: "2.0.0",
    description: "Pattern-based reasoning with templates for common inquiries",
    
    config: {
        executionStrategy: "reasoning",
        resourceSubType: "CustomerSupport",
        estimatedDuration: "15-25 seconds",
        estimatedCost: "$0.06-0.08 per inquiry",
        
        // Reasoning strategy improvements
        reasoning: {
            approach: "Pattern matching with reasoning fallback",
            patterns: 12, // Number of identified common patterns
            templateCoverage: "60% of inquiries",
            contextOptimization: "Selective context retrieval"
        }
    },
    
    // Optimized execution flow
    executionSteps: [
        {
            name: "Pattern Classification",
            prompt: "Classify inquiry into: billing, technical, account, shipping, or custom.",
            expectedTime: "2-3s"
        },
        {
            name: "Template Selection",
            prompt: "If pattern matches (confidence > 0.8), select appropriate response template.",
            expectedTime: "1s"
        },
        {
            name: "Contextual Adaptation",
            prompt: "Adapt template with customer-specific details and current context.",
            expectedTime: "5-10s"
        },
        {
            name: "Fallback Reasoning",
            prompt: "For non-pattern inquiries, use focused reasoning on specific aspects.",
            expectedTime: "10-15s",
            conditional: true
        }
    ],
    
    // Improved performance metrics
    metrics: {
        avgExecutionTime: "18 seconds",
        avgCost: "$0.07",
        accuracy: "95%",
        customerSatisfaction: "88%",
        patternHitRate: "62%",
        fallbackRate: "38%"
    },
    
    // Evolution insights
    evolutionInsights: {
        costReduction: "50%",
        speedImprovement: "65%",
        accuracyGain: "3.3%",
        
        learnedPatterns: [
            "Password reset requests (15%)",
            "Order status inquiries (12%)",
            "Billing questions (10%)",
            "Technical troubleshooting (8%)",
            "Account updates (7%)",
            "Shipping delays (5%)",
            "Return requests (3%)"
        ],
        
        nextOptimizations: [
            "Direct API calls for data retrieval",
            "Pre-computed responses for top patterns",
            "Automated action execution"
        ]
    }
};