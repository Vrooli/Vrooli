/**
 * MOISE+ Type Definitions
 * 
 * Complete type system for MOISE+ organizational modeling
 * Supporting structural, functional, and normative specifications
 */

import type { AgentCapability as BaseAgentCapability } from "@vrooli/shared";

/**
 * Extended agent capabilities for test fixtures
 * Includes base capabilities plus domain-specific ones
 */
export type ExtendedAgentCapability = BaseAgentCapability
    // Strategic & Leadership
    | "strategic_planning"
    | "leadership"
    | "risk_assessment"
    // Medical & Healthcare
    | "medical_expertise"
    | "clinical_standards"
    | "patient_safety"
    | "treatment_validation"
    | "prescription_management"
    // Compliance & Regulatory
    | "regulatory_expertise"
    | "hipaa_compliance"
    | "privacy_compliance"
    | "compliance_monitoring"
    | "audit_management"
    // Quality & Security
    | "quality_assurance"
    | "security_monitoring"
    | "threat_detection"
    | "fraud_detection"
    // Technical & Trading
    | "pattern_recognition"
    | "algorithm_optimization"
    | "trade_validation"
    | "quantitative_analysis"
    | "market_monitoring"
    // Research & Analysis
    | "research_methodology"
    | "data_analysis"
    | "experimental_design"
    | "publication_management"
    | "peer_review"
    | "knowledge_synthesis";

/**
 * Complete MOISE+ organizational specification
 */
export interface MOISEPlusSpecification {
    structural: StructuralSpecification;
    functional: FunctionalSpecification;
    normative: NormativeSpecification;
}

/**
 * Structural specification - defines roles, groups, and relationships
 */
export interface StructuralSpecification {
    roles: RoleSpecification[];
    groups: GroupSpecification[];
    links: LinkSpecification[];
    compatibilities: CompatibilityRule[];
    inheritance: InheritanceRule[];
}

/**
 * Role specification with MOISE+ properties
 */
export interface RoleSpecification {
    id: string;
    name: string;
    description: string;
    minInstances: number;
    maxInstances: number;
    inheritance?: string[]; // Parent roles
    requiredCapabilities: ExtendedAgentCapability[];
    responsibilities: Responsibility[];
    communicationLinks: CommunicationLink[];
}

/**
 * Responsibility definition
 */
export interface Responsibility {
    id: string;
    description: string;
    priority: "critical" | "high" | "medium" | "low";
    frequency: "continuous" | "periodic" | "event-driven" | "on-demand";
}

/**
 * Communication link between roles
 */
export interface CommunicationLink {
    type: "authority" | "coordination" | "collaboration" | "information";
    target: string[]; // Target role IDs
    bidirectional: boolean;
}

/**
 * Group specification for organizational units
 */
export interface GroupSpecification {
    id: string;
    name: string;
    type: "team" | "functional_unit" | "division" | "committee";
    roles: string[]; // Role IDs allowed in this group
    minSize: number;
    maxSize: number;
    subgroups?: string[]; // Child group IDs
    parentGroup?: string; // Parent group ID
}

/**
 * Link specification between groups
 */
export interface LinkSpecification {
    from: string; // Group ID
    to: string; // Group ID
    type: "composition" | "aggregation" | "association";
    cardinality: string; // e.g., "1..*", "0..1"
}

/**
 * Compatibility rule between roles
 */
export interface CompatibilityRule {
    role1: string;
    role2: string;
    type: "mutex" | "compatible" | "requires";
    reason: string;
}

/**
 * Inheritance rule for roles
 */
export interface InheritanceRule {
    parent: string;
    child: string;
    inheritedProperties: ("capabilities" | "responsibilities" | "norms")[];
}

/**
 * Functional specification - defines goals, plans, and missions
 */
export interface FunctionalSpecification {
    goals: GoalSpecification[];
    plans: PlanSpecification[];
    missions: MissionSpecification[];
    schemes: SchemeSpecification[];
}

/**
 * Goal specification with decomposition
 */
export interface GoalSpecification {
    id: string;
    type: "achievement" | "maintenance" | "optimization";
    description: string;
    decomposition?: "AND" | "OR" | "XOR";
    subgoals?: string[]; // Subgoal IDs
    successCriteria: SuccessCriteria;
    deadline?: string;
}

/**
 * Success criteria for goals
 */
export interface SuccessCriteria {
    metric: string;
    threshold: number;
    operator?: ">" | ">=" | "<" | "<=" | "==" | "!=";
    timeWindow?: string;
}

/**
 * Plan specification for achieving goals
 */
export interface PlanSpecification {
    id: string;
    name: string;
    goals: string[]; // Goal IDs this plan achieves
    steps: PlanStep[];
    preconditions?: string[];
    postconditions?: string[];
}

/**
 * Individual step in a plan
 */
export interface PlanStep {
    id: string;
    action: string;
    responsibleRole?: string;
    duration?: string;
    dependencies?: string[]; // Other step IDs
}

/**
 * Mission specification - assigns goals to agents
 */
export interface MissionSpecification {
    id: string;
    name: string;
    goals: string[]; // Goal IDs
    minAgents: number;
    maxAgents: number;
    preferredRoles?: string[]; // Role IDs
    deadline?: string;
}

/**
 * Scheme specification - organizational context
 */
export interface SchemeSpecification {
    id: string;
    name: string;
    rootGoal: string; // Main goal ID
    missions: string[]; // Mission IDs
    monitoringScheme?: string; // How progress is monitored
}

/**
 * Normative specification - defines rules and constraints
 */
export interface NormativeSpecification {
    norms: NormSpecification[];
    permissions: PermissionRule[];
    obligations: ObligationRule[];
    prohibitions: ProhibitionRule[];
}

/**
 * Base norm specification
 */
export interface NormSpecification {
    id: string;
    type: "obligation" | "permission" | "prohibition";
    scope: string | "all_roles"; // Role ID or global
    condition: string;
    target: string; // Action or goal
    deadline?: string;
    sanction?: string;
    priority?: "critical" | "high" | "medium" | "low";
}

/**
 * Permission rule
 */
export interface PermissionRule extends NormSpecification {
    type: "permission";
    exceptions?: string[]; // Conditions where permission doesn't apply
}

/**
 * Obligation rule
 */
export interface ObligationRule extends NormSpecification {
    type: "obligation";
    fulfillmentMonitoring?: "continuous" | "periodic" | "event-based";
}

/**
 * Prohibition rule
 */
export interface ProhibitionRule extends NormSpecification {
    type: "prohibition";
    violationDetection?: "immediate" | "periodic" | "reported";
}

/**
 * Complete MOISE+ organization
 */
export interface MOISEPlusOrganization {
    id: string;
    name: string;
    description: string;
    specification: MOISEPlusSpecification;
    activeSchemes?: SchemeSpecification[];
    performanceMetrics?: OrganizationMetrics;
}

/**
 * Organization performance metrics
 */
export interface OrganizationMetrics {
    goalCompletionRate: number;
    normComplianceRate: number;
    averageMissionDuration: number;
    resourceEfficiency: number;
    collaborationScore: number;
}

/**
 * Helper type for norm conditions
 */
export type NormCondition = 
    | "always"
    | "weekly_review"
    | "resource_allocation"
    | "performance_below_threshold"
    | "phi_breach_detected"
    | "goal_deadline_approaching"
    | string; // Custom conditions

/**
 * Helper type for norm actions
 */
export type NormAction = 
    | "submit_progress_report"
    | "ensure_fair_distribution"
    | "adjust_goals"
    | "allocate_beyond_budget"
    | "report_to_compliance_board"
    | string; // Custom actions
