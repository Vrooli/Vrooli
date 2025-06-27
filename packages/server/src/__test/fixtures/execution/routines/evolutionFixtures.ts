/**
 * Evolution Fixtures for Routine Development
 * 
 * These fixtures demonstrate how routines evolve from conversational to deterministic,
 * showing measurable improvements in performance, accuracy, and cost efficiency.
 */

import type { RoutineConfigObject } from "@vrooli/shared";

/**
 * Technical Support Specialist - Version 3 (Conversational Stage)
 * Handles technical inquiries with AI-driven conversation
 */
export const TECHNICAL_SUPPORT_SPECIALIST_V3: RoutineConfigObject = {
    __version: "3.0.0",
    routineType: "conversational",
    name: "Technical Support Specialist V3",
    description: "Advanced technical support with conversational AI capabilities",
    steps: [
        {
            id: "analyze_issue",
            name: "Analyze Technical Issue",
            type: "conversational",
            inputs: ["user_query", "system_context"],
            outputs: ["issue_analysis", "complexity_assessment"],
        },
        {
            id: "provide_solution", 
            name: "Provide Solution",
            type: "conversational",
            inputs: ["issue_analysis", "knowledge_base"],
            outputs: ["solution_steps", "confidence_score"],
        },
    ],
    emergenceConfig: {
        capabilities: ["technical_diagnosis", "solution_generation", "customer_communication"],
        learningEnabled: true,
        adaptationThreshold: 0.8,
    },
};

/**
 * Billing Specialist - Version 3 (Reasoning Stage)
 * Handles billing inquiries with reasoning-based problem solving
 */
export const BILLING_SPECIALIST_V3: RoutineConfigObject = {
    __version: "3.0.0", 
    routineType: "reasoning",
    name: "Billing Specialist V3",
    description: "Billing support with advanced reasoning capabilities",
    steps: [
        {
            id: "analyze_billing_query",
            name: "Analyze Billing Query",
            type: "reasoning",
            inputs: ["billing_query", "account_data"],
            outputs: ["query_classification", "required_actions"],
        },
        {
            id: "resolve_billing_issue",
            name: "Resolve Billing Issue", 
            type: "reasoning",
            inputs: ["query_classification", "billing_rules"],
            outputs: ["resolution_plan", "estimated_time"],
        },
    ],
    emergenceConfig: {
        capabilities: ["billing_analysis", "policy_reasoning", "resolution_planning"],
        learningEnabled: true,
        adaptationThreshold: 0.85,
    },
};

/**
 * Account Specialist - Version 3 (Deterministic Stage)
 * Handles account management with deterministic workflows
 */
export const ACCOUNT_SPECIALIST_V3: RoutineConfigObject = {
    __version: "3.0.0",
    routineType: "deterministic", 
    name: "Account Specialist V3",
    description: "Account management with optimized deterministic processes",
    steps: [
        {
            id: "validate_account",
            name: "Validate Account",
            type: "deterministic",
            inputs: ["account_id", "verification_data"],
            outputs: ["validation_result", "account_status"],
        },
        {
            id: "process_account_request",
            name: "Process Account Request",
            type: "deterministic", 
            inputs: ["validation_result", "request_type"],
            outputs: ["processing_result", "next_steps"],
        },
    ],
    emergenceConfig: {
        capabilities: ["account_validation", "request_processing", "workflow_optimization"],
        learningEnabled: false, // Deterministic stage
        adaptationThreshold: 0.95,
    },
};

/**
 * Escalation Handler - Version 2 (Reasoning Stage)
 * Manages complex escalations requiring reasoning
 */
export const ESCALATION_HANDLER_V2: RoutineConfigObject = {
    __version: "2.0.0",
    routineType: "reasoning",
    name: "Escalation Handler V2", 
    description: "Complex escalation management with reasoning capabilities",
    steps: [
        {
            id: "assess_escalation",
            name: "Assess Escalation",
            type: "reasoning",
            inputs: ["escalation_data", "priority_matrix"],
            outputs: ["urgency_level", "routing_decision"],
        },
        {
            id: "route_escalation",
            name: "Route Escalation",
            type: "reasoning",
            inputs: ["urgency_level", "available_agents"],
            outputs: ["assigned_agent", "estimated_resolution"],
        },
    ],
    emergenceConfig: {
        capabilities: ["escalation_assessment", "intelligent_routing", "priority_management"],
        learningEnabled: true,
        adaptationThreshold: 0.9,
    },
};

/**
 * Response Synthesizer - Version 1 (Routing Stage)
 * Routes and synthesizes responses from multiple sources
 */
export const RESPONSE_SYNTHESIZER_V1: RoutineConfigObject = {
    __version: "1.0.0",
    routineType: "routing",
    name: "Response Synthesizer V1",
    description: "Intelligent response synthesis and routing",
    steps: [
        {
            id: "collect_responses",
            name: "Collect Responses",
            type: "routing",
            inputs: ["source_responses", "context_data"],
            outputs: ["response_collection", "quality_scores"],
        },
        {
            id: "synthesize_final_response",
            name: "Synthesize Final Response",
            type: "routing",
            inputs: ["response_collection", "synthesis_rules"],
            outputs: ["final_response", "confidence_level"],
        },
    ],
    emergenceConfig: {
        capabilities: ["response_collection", "intelligent_synthesis", "quality_assessment"],
        learningEnabled: true,
        adaptationThreshold: 0.75,
    },
};
