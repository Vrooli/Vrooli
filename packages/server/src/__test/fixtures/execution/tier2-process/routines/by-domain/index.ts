/**
 * Routine fixtures index
 * 
 * Aggregates all routine types and provides helper functions
 */

import { ResourceSubType } from "@vrooli/shared";
import type { 
    RoutineFixture, 
    RoutineFixtureCollection, 
    RoutineCategory, 
    ExecutionStrategy,
    RoutineStats,
    AgentRoutineMapping,
} from "./types.js";
import { SECURITY_ROUTINES } from "./securityRoutines.js";
import { MEDICAL_ROUTINES } from "./medicalRoutines.js";
import { PERFORMANCE_ROUTINES } from "./performanceRoutines.js";
import { SYSTEM_ROUTINES } from "./systemRoutines.js";
import { BPMN_WORKFLOWS } from "./bpmnWorkflows.js";
import { API_BOOTSTRAP_ROUTINES } from "./apiBootstrapRoutines.js";
import { DOCUMENT_BOOTSTRAP_ROUTINES, DATA_TRANSFORMATION_ROUTINES } from "./dataBootstrapRoutines.js";

/**
 * All sequential (single-step) routines
 */
export const SEQUENTIAL_ROUTINES = {
    ...SECURITY_ROUTINES,
    ...MEDICAL_ROUTINES,
    ...PERFORMANCE_ROUTINES,
    ...SYSTEM_ROUTINES,
};

/**
 * All bootstrap routines (API and data)
 */
export const BOOTSTRAP_ROUTINES = {
    ...API_BOOTSTRAP_ROUTINES,
    ...DOCUMENT_BOOTSTRAP_ROUTINES,
    ...DATA_TRANSFORMATION_ROUTINES,
};

/**
 * All BPMN-based multi-step workflows
 */
export const BPMN_ROUTINES = BPMN_WORKFLOWS;

/**
 * Re-export individual routine categories
 */
export {
    SECURITY_ROUTINES,
    MEDICAL_ROUTINES,
    PERFORMANCE_ROUTINES,
    SYSTEM_ROUTINES,
    BPMN_WORKFLOWS,
    API_BOOTSTRAP_ROUTINES,
    DOCUMENT_BOOTSTRAP_ROUTINES,
    DATA_TRANSFORMATION_ROUTINES,
};

/**
 * Helper function to get all routines as a flat array
 */
export function getAllRoutines(): RoutineFixture[] {
    return [
        ...Object.values(SEQUENTIAL_ROUTINES),
        ...Object.values(BPMN_ROUTINES),
        ...Object.values(BOOTSTRAP_ROUTINES),
    ];
}

/**
 * Helper function to get routine by ID
 */
export function getRoutineById(routineId: string): RoutineFixture | undefined {
    return getAllRoutines().find(routine => routine.id === routineId);
}

/**
 * Helper function to get routines by execution strategy
 */
export function getRoutinesByStrategy(strategy: ExecutionStrategy): RoutineFixture[] {
    return getAllRoutines().filter(routine => 
        routine.config.executionStrategy === strategy,
    );
}

/**
 * Helper function to get routines by resource subtype
 */
export function getRoutinesByResourceSubType(subType: ResourceSubType): RoutineFixture[] {
    return getAllRoutines().filter(routine => 
        routine.resourceSubType === subType,
    );
}

/**
 * Helper function to get routines by category
 */
export function getRoutinesByCategory(category: RoutineCategory): RoutineFixture[] {
    switch (category) {
        case "security":
            return Object.values(SECURITY_ROUTINES);
        case "medical":
            return Object.values(MEDICAL_ROUTINES);
        case "performance":
            return Object.values(PERFORMANCE_ROUTINES);
        case "system":
            return Object.values(SYSTEM_ROUTINES);
        case "bpmn":
            return Object.values(BPMN_ROUTINES);
        case "api_bootstrap":
            return Object.values(API_BOOTSTRAP_ROUTINES);
        case "document_bootstrap":
            return Object.values(DOCUMENT_BOOTSTRAP_ROUTINES);
        case "data_transformation":
            return Object.values(DATA_TRANSFORMATION_ROUTINES);
        default:
            return [];
    }
}

/**
 * Map of agent IDs to their associated routine IDs
 */
export const AGENT_ROUTINE_MAP: AgentRoutineMapping = {
    // Security agents
    "hipaa_compliance_monitor": [SECURITY_ROUTINES.HIPAA_COMPLIANCE_CHECK.id],
    "medical_ai_safety_monitor": [MEDICAL_ROUTINES.MEDICAL_DIAGNOSIS_VALIDATION.id],
    "trading_fraud_detector": [SECURITY_ROUTINES.TRADING_PATTERN_ANALYSIS.id],
    "api_security_monitor": [SECURITY_ROUTINES.API_SECURITY_SCAN.id],
    "data_privacy_compliance": [SECURITY_ROUTINES.GDPR_DATA_AUDIT.id],
    
    // Resilience agents
    "resilience_pattern_learner": [BPMN_ROUTINES.SYSTEM_FAILURE_ANALYSIS_WORKFLOW.id],
    
    // Strategy evolution agents
    "routine_performance_analyzer": [PERFORMANCE_ROUTINES.PERFORMANCE_BOTTLENECK_DETECTION.id],
    "cost_optimization_specialist": [PERFORMANCE_ROUTINES.COST_ANALYSIS.id],
    
    // Quality agents
    "output_quality_monitor": [PERFORMANCE_ROUTINES.OUTPUT_QUALITY_ASSESSMENT.id],
    
    // Monitoring agents
    "system_health_monitor": [SYSTEM_ROUTINES.SYSTEM_HEALTH_CHECK.id],
    
    // Swarm routines (multi-agent workflows)
    "healthcare_security_swarm": [BPMN_ROUTINES.MEDICAL_TREATMENT_VALIDATION.id],
    "security_swarm": [BPMN_ROUTINES.COMPREHENSIVE_SECURITY_AUDIT.id],
    "resilience_evolution_swarm": [BPMN_ROUTINES.RESILIENCE_OPTIMIZATION_WORKFLOW.id],
    
    // API bootstrapping agents
    "api_integration_creator": [API_BOOTSTRAP_ROUTINES.STRIPE_INTEGRATION_BOOTSTRAP.id, API_BOOTSTRAP_ROUTINES.TWILIO_SMS_BOOTSTRAP.id],
    "api_test_generator": [API_BOOTSTRAP_ROUTINES.SENDGRID_EMAIL_BOOTSTRAP.id],
    
    // Document bootstrapping agents
    "intelligent_document_creator": [DOCUMENT_BOOTSTRAP_ROUTINES.QUARTERLY_REPORT_GENERATOR.id, DOCUMENT_BOOTSTRAP_ROUTINES.PRESENTATION_BUILDER.id],
    "format_optimization_specialist": [DATA_TRANSFORMATION_ROUTINES.MULTI_FORMAT_CONVERTER.id],
};

/**
 * Get routines associated with a specific agent
 */
export function getRoutinesForAgent(agentId: string): RoutineFixture[] {
    const routineIds = AGENT_ROUTINE_MAP[agentId] || [];
    return routineIds
        .map(id => getRoutineById(id))
        .filter((routine): routine is RoutineFixture => routine !== undefined);
}

/**
 * Routine statistics
 */
export function getRoutineStats(): RoutineStats {
    const sequential = Object.values(SEQUENTIAL_ROUTINES);
    const bpmn = Object.values(BPMN_ROUTINES);
    const bootstrap = Object.values(BOOTSTRAP_ROUTINES);
    
    return {
        total: sequential.length + bpmn.length + bootstrap.length,
        sequential: sequential.length,
        bpmn: bpmn.length,
        bootstrap: bootstrap.length,
        byCategory: {
            security: Object.values(SECURITY_ROUTINES).length,
            medical: Object.values(MEDICAL_ROUTINES).length,
            performance: Object.values(PERFORMANCE_ROUTINES).length,
            system: Object.values(SYSTEM_ROUTINES).length,
            api_bootstrap: Object.values(API_BOOTSTRAP_ROUTINES).length,
            document_bootstrap: Object.values(DOCUMENT_BOOTSTRAP_ROUTINES).length,
            data_transformation: Object.values(DATA_TRANSFORMATION_ROUTINES).length,
        },
        byStrategy: {
            reasoning: getRoutinesByStrategy("reasoning").length,
            deterministic: getRoutinesByStrategy("deterministic").length,
            conversational: getRoutinesByStrategy("conversational").length,
            auto: getRoutinesByStrategy("auto").length,
        },
        byResourceType: {
            action: getRoutinesByResourceSubType(ResourceSubType.RoutineInternalAction).length,
            generate: getRoutinesByResourceSubType(ResourceSubType.RoutineGenerate).length,
            code: getRoutinesByResourceSubType(ResourceSubType.RoutineCode).length,
            web: getRoutinesByResourceSubType(ResourceSubType.RoutineWeb).length,
            multiStep: getRoutinesByResourceSubType(ResourceSubType.RoutineMultiStep).length,
        },
    };
}
