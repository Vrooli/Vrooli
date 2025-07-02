/**
 * Core type definitions for security and safety
 * These types define the security framework across all tiers
 */

/**
 * Security levels
 */
export enum SecurityLevel {
    PUBLIC = "PUBLIC",
    PRIVATE = "PRIVATE",
    RESTRICTED = "RESTRICTED",
    CONFIDENTIAL = "CONFIDENTIAL"
}

/**
 * Data sensitivity levels for context validation and protection
 */
export enum DataSensitivity {
    PUBLIC = "public",
    INTERNAL = "internal", 
    CONFIDENTIAL = "confidential",
    SECRET = "secret",
    PII = "pii"
}

/**
 * Guard rail types
 */
export enum GuardRailType {
    INPUT_VALIDATION = "INPUT_VALIDATION",
    OUTPUT_VALIDATION = "OUTPUT_VALIDATION",
    RESOURCE_LIMIT = "RESOURCE_LIMIT",
    ACCESS_CONTROL = "ACCESS_CONTROL",
    CONTENT_FILTER = "CONTENT_FILTER",
    RATE_LIMIT = "RATE_LIMIT",
    COMPLIANCE = "COMPLIANCE"
}

/**
 * Guard rail definition
 */
export interface GuardRail {
    id: string;
    type: GuardRailType;
    name: string;
    description: string;
    enabled: boolean;
    config: GuardRailConfig;
    actions: GuardRailAction[];
}

/**
 * Guard rail configuration
 */
export interface GuardRailConfig {
    severity: "info" | "warning" | "error" | "critical";
    mode: "monitor" | "block" | "modify";
    rules: GuardRailRule[];
    exceptions?: string[];
}

/**
 * Guard rail rule
 */
export interface GuardRailRule {
    id: string;
    condition: string; // Expression to evaluate
    message: string;
    metadata?: Record<string, unknown>;
}

/**
 * Guard rail action
 */
export interface GuardRailAction {
    type: "log" | "alert" | "block" | "modify" | "notify";
    config: Record<string, unknown>;
}

/**
 * Guard rail violation
 */
export interface GuardRailViolation {
    id: string;
    timestamp: Date;
    guardRailId: string;
    ruleId: string;
    severity: string;
    message: string;
    context: ViolationContext;
    action: string;
    resolved: boolean;
}

/**
 * Violation context
 */
export interface ViolationContext {
    tier: 1 | 2 | 3;
    component: string;
    userId?: string;
    requestId: string;
    input?: unknown;
    output?: unknown;
    metadata: Record<string, unknown>;
}

/**
 * Barrier synchronization point
 */
export interface Barrier {
    id: string;
    name: string;
    description: string;
    participants: BarrierParticipant[];
    condition: BarrierCondition;
    timeout: number;
    state: BarrierState;
}

/**
 * Barrier participant
 */
export interface BarrierParticipant {
    id: string;
    type: "tier" | "component" | "agent";
    required: boolean;
    status: "waiting" | "ready" | "passed" | "failed";
    signalTime?: Date;
}

/**
 * Barrier condition
 */
export interface BarrierCondition {
    type: "all" | "any" | "threshold" | "custom";
    threshold?: number; // For threshold type
    expression?: string; // For custom type
}

/**
 * Barrier state
 */
export interface BarrierState {
    status: "waiting" | "satisfied" | "timeout" | "failed";
    startTime: Date;
    endTime?: Date;
    result?: Record<string, unknown>;
}

/**
 * Emergency stop configuration
 */
export interface EmergencyStopConfig {
    enabled: boolean;
    triggers: EmergencyTrigger[];
    actions: EmergencyAction[];
    notificationChannels: NotificationChannel[];
}

/**
 * Emergency trigger
 */
export interface EmergencyTrigger {
    id: string;
    type: "manual" | "automatic";
    condition: string;
    severity: "high" | "critical";
    description: string;
}

/**
 * Emergency action
 */
export interface EmergencyAction {
    type: "stop_all" | "stop_tier" | "stop_component" | "pause" | "isolate";
    target?: string;
    preserveState: boolean;
    notifyUsers: boolean;
}

/**
 * Notification channel
 */
export interface NotificationChannel {
    type: "email" | "sms" | "webhook" | "console";
    config: Record<string, unknown>;
    recipients: string[];
}

/**
 * Access control policy
 */
export interface AccessPolicy {
    id: string;
    name: string;
    description: string;
    subjects: Subject[];
    resources: SecurityResource[];
    actions: string[];
    conditions?: AccessCondition[];
    effect: "allow" | "deny";
}

/**
 * Subject in access control
 */
export interface Subject {
    type: "user" | "role" | "group" | "service";
    id: string;
    attributes?: Record<string, unknown>;
}

/**
 * Security resource in access control
 * Represents a protected resource for permission checks
 */
export interface SecurityResource {
    type: string;
    id: string;
    attributes?: Record<string, unknown>;
}

/**
 * Access condition
 */
export interface AccessCondition {
    type: "time" | "location" | "attribute" | "custom";
    operator: string;
    value: unknown;
}

/**
 * Compliance requirement
 */
export interface ComplianceRequirement {
    id: string;
    name: string;
    standard: string; // e.g., "GDPR", "HIPAA", "SOC2"
    requirements: string[];
    checks: ComplianceCheck[];
    status: ComplianceStatus;
}

/**
 * Compliance check
 */
export interface ComplianceCheck {
    id: string;
    name: string;
    type: "automated" | "manual";
    frequency: string; // e.g., "realtime", "daily", "weekly"
    lastChecked?: Date;
    result?: "passed" | "failed" | "partial";
}

/**
 * Compliance status
 */
export interface ComplianceStatus {
    compliant: boolean;
    score: number; // 0-100
    issues: ComplianceIssue[];
    lastAudit?: Date;
    nextAudit?: Date;
}

/**
 * Compliance issue
 */
export interface ComplianceIssue {
    id: string;
    checkId: string;
    severity: "low" | "medium" | "high" | "critical";
    description: string;
    remediation: string;
    deadline?: Date;
    resolved: boolean;
}

/**
 * Threat detection result
 */
export interface ThreatDetection {
    id: string;
    timestamp: Date;
    type: "injection" | "overflow" | "privilege_escalation" | "data_leak" | "dos" | "other";
    confidence: number; // 0-1
    severity: "low" | "medium" | "high" | "critical";
    source: string;
    target: string;
    evidence: Record<string, unknown>;
    mitigated: boolean;
}

/**
 * Security audit trail
 */
export interface SecurityAudit {
    id: string;
    timestamp: Date;
    actor: string;
    action: string;
    resource: string;
    result: "success" | "failure" | "partial";
    metadata: Record<string, unknown>;
    ipAddress?: string;
    userAgent?: string;
}

/**
 * Security context for execution
 */
export interface SecurityContext {
    level: SecurityLevel;
    policies: string[]; // Policy IDs
    guardRails: string[]; // Guard rail IDs
    barriers: string[]; // Barrier IDs
    compliance: string[]; // Compliance requirement IDs
    audit: boolean;
}

/**
 * Security validation result
 */
export interface SecurityValidation {
    valid: boolean;
    violations: GuardRailViolation[];
    warnings: string[];
    metadata: Record<string, unknown>;
}

/**
 * Enhanced security context for tier communication
 */
export interface EnhancedSecurityContext extends SecurityContext {
    tier: 1 | 2 | 3;
    requestId: string;
    sessionId?: string;
    origin: SecurityOrigin;
    permissions: Permission[];
    riskScore: number; // 0-1
    threatLevel: ThreatLevel;
    validationResults: SecurityValidation[];
}

/**
 * Security origin information
 */
export interface SecurityOrigin {
    type: OriginType;
    identifier: string;
    verified: boolean;
    trustLevel: TrustLevel;
    metadata: Record<string, unknown>;
}

/**
 * Origin types
 */
export enum OriginType {
    USER = "USER",
    AGENT = "AGENT",
    TIER = "TIER",
    EXTERNAL_API = "EXTERNAL_API",
    SYSTEM = "SYSTEM"
}

/**
 * Trust levels
 */
export enum TrustLevel {
    UNTRUSTED = "UNTRUSTED",
    LOW = "LOW",
    MEDIUM = "MEDIUM",
    HIGH = "HIGH",
    SYSTEM = "SYSTEM"
}

/**
 * Threat levels
 */
export enum ThreatLevel {
    NONE = "NONE",
    LOW = "LOW",
    MEDIUM = "MEDIUM",
    HIGH = "HIGH",
    CRITICAL = "CRITICAL"
}

/**
 * Permission definition
 */
export interface Permission {
    action: string;
    resource: string;
    conditions?: PermissionCondition[];
    scope?: PermissionScope;
    temporary?: boolean;
    expiresAt?: Date;
}

/**
 * Permission condition
 */
export interface PermissionCondition {
    type: string;
    operator: string;
    value: unknown;
    description: string;
}

/**
 * Permission scope
 */
export interface PermissionScope {
    tier?: number;
    component?: string;
    operation?: string;
    dataTypes?: string[];
    timeWindow?: TimeWindow;
}

/**
 * Time window for permissions
 */
export interface TimeWindow {
    start?: Date;
    end?: Date;
    duration?: number;
    recurring?: boolean;
}

/**
 * AI security validation
 */
export interface AISecurityValidation {
    inputValidation: ValidationResult;
    outputValidation: ValidationResult;
    promptInjectionCheck: ValidationResult;
    contentSafetyCheck: ValidationResult;
    biasCheck: ValidationResult;
    privacyCheck: ValidationResult;
    overallRisk: number; // 0-1
}

/**
 * Validation result
 */
export interface ValidationResult {
    passed: boolean;
    confidence: number; // 0-1
    issues: ValidationIssue[];
    recommendation: string;
    metadata: Record<string, unknown>;
}

/**
 * Validation issue
 */
export interface ValidationIssue {
    type: string;
    severity: "low" | "medium" | "high" | "critical";
    description: string;
    location?: string;
    suggestion?: string;
}

/**
 * Security incident
 */
export interface SecurityIncident {
    id: string;
    timestamp: Date;
    type: IncidentType;
    severity: "low" | "medium" | "high" | "critical";
    status: IncidentStatus;
    title: string;
    description: string;
    affected: AffectedEntity[];
    detection: DetectionMethod;
    response: IncidentResponse;
    resolution?: IncidentResolution;
    metadata: Record<string, unknown>;
}

/**
 * Incident types
 */
export enum IncidentType {
    UNAUTHORIZED_ACCESS = "UNAUTHORIZED_ACCESS",
    DATA_BREACH = "DATA_BREACH",
    PRIVILEGE_ESCALATION = "PRIVILEGE_ESCALATION",
    INJECTION_ATTACK = "INJECTION_ATTACK",
    DENIAL_OF_SERVICE = "DENIAL_OF_SERVICE",
    MALWARE = "MALWARE",
    SOCIAL_ENGINEERING = "SOCIAL_ENGINEERING",
    INSIDER_THREAT = "INSIDER_THREAT",
    CONFIGURATION_ERROR = "CONFIGURATION_ERROR",
    OTHER = "OTHER"
}

/**
 * Incident status
 */
export enum IncidentStatus {
    DETECTED = "DETECTED",
    INVESTIGATING = "INVESTIGATING",
    CONTAINED = "CONTAINED",
    RESOLVED = "RESOLVED",
    CLOSED = "CLOSED"
}

/**
 * Affected entity
 */
export interface AffectedEntity {
    type: EntityType;
    identifier: string;
    impact: ImpactLevel;
    compromised: boolean;
    containmentActions: string[];
}

/**
 * Entity types
 */
export enum EntityType {
    USER = "USER",
    SYSTEM = "SYSTEM",
    DATA = "DATA",
    SERVICE = "SERVICE",
    NETWORK = "NETWORK"
}

/**
 * Impact levels
 */
export enum ImpactLevel {
    NONE = "NONE",
    LOW = "LOW",
    MEDIUM = "MEDIUM",
    HIGH = "HIGH",
    CRITICAL = "CRITICAL"
}

/**
 * Detection method
 */
export interface DetectionMethod {
    type: DetectionType;
    source: string;
    confidence: number; // 0-1
    automatedResponse: boolean;
    alertGenerated: boolean;
}

/**
 * Detection types
 */
export enum DetectionType {
    AUTOMATED = "AUTOMATED",
    MANUAL = "MANUAL",
    USER_REPORT = "USER_REPORT",
    THIRD_PARTY = "THIRD_PARTY",
    ROUTINE_AUDIT = "ROUTINE_AUDIT"
}

/**
 * Incident response
 */
export interface IncidentResponse {
    startTime: Date;
    responder: string;
    actions: ResponseAction[];
    timeline: ResponseTimeline[];
    escalated: boolean;
    escalationReason?: string;
}

/**
 * Response action
 */
export interface ResponseAction {
    type: ResponseActionType;
    description: string;
    timestamp: Date;
    performer: string;
    result: ActionResult;
    metadata: Record<string, unknown>;
}

/**
 * Response action types
 */
export enum ResponseActionType {
    ISOLATE = "ISOLATE",
    QUARANTINE = "QUARANTINE",
    BLOCK = "BLOCK",
    MONITOR = "MONITOR",
    COLLECT_EVIDENCE = "COLLECT_EVIDENCE",
    NOTIFY = "NOTIFY",
    PATCH = "PATCH",
    RESTORE = "RESTORE",
    INVESTIGATE = "INVESTIGATE"
}

/**
 * Action result
 */
export interface ActionResult {
    success: boolean;
    impact: string;
    evidence?: string[];
    nextSteps?: string[];
}

/**
 * Response timeline
 */
export interface ResponseTimeline {
    timestamp: Date;
    event: string;
    details: string;
    performer: string;
}

/**
 * Incident resolution
 */
export interface IncidentResolution {
    timestamp: Date;
    method: ResolutionMethod;
    rootCause: string;
    preventionMeasures: string[];
    lessonsLearned: string[];
    followUpRequired: boolean;
    verificationSteps: string[];
}

/**
 * Resolution methods
 */
export enum ResolutionMethod {
    AUTOMATED = "AUTOMATED",
    MANUAL = "MANUAL",
    HYBRID = "HYBRID",
    EXTERNAL_ASSISTANCE = "EXTERNAL_ASSISTANCE"
}

/**
 * Security audit enhanced
 */
export interface EnhancedSecurityAudit extends SecurityAudit {
    risk: SecurityRisk;
    compliance: ComplianceStatus;
    recommendations: AuditRecommendation[];
}

/**
 * Security risk assessment
 */
export interface SecurityRisk {
    level: RiskLevel;
    factors: RiskFactor[];
    mitigations: RiskMitigation[];
    residualRisk: number; // 0-1
    acceptanceStatus: RiskAcceptanceStatus;
}

/**
 * Risk levels
 */
export enum RiskLevel {
    VERY_LOW = "VERY_LOW",
    LOW = "LOW",
    MEDIUM = "MEDIUM",
    HIGH = "HIGH",
    VERY_HIGH = "VERY_HIGH"
}

/**
 * Risk factor
 */
export interface RiskFactor {
    type: string;
    impact: number; // 0-1
    likelihood: number; // 0-1
    description: string;
    mitigated: boolean;
}

/**
 * Risk mitigation
 */
export interface RiskMitigation {
    id: string;
    description: string;
    effectiveness: number; // 0-1
    cost: CostLevel;
    implemented: boolean;
    implementationDate?: Date;
}

/**
 * Cost levels
 */
export enum CostLevel {
    VERY_LOW = "VERY_LOW",
    LOW = "LOW",
    MEDIUM = "MEDIUM",
    HIGH = "HIGH",
    VERY_HIGH = "VERY_HIGH"
}

/**
 * Risk acceptance status
 */
export enum RiskAcceptanceStatus {
    PENDING = "PENDING",
    ACCEPTED = "ACCEPTED",
    REJECTED = "REJECTED",
    REQUIRES_REVIEW = "REQUIRES_REVIEW"
}

/**
 * Audit recommendation
 */
export interface AuditRecommendation {
    id: string;
    priority: RecommendationPriority;
    category: string;
    description: string;
    implementation: string;
    timeline: string;
    resources: string[];
    impact: string;
    status: RecommendationStatus;
}

/**
 * Recommendation priorities
 */
export enum RecommendationPriority {
    CRITICAL = "CRITICAL",
    HIGH = "HIGH",
    MEDIUM = "MEDIUM",
    LOW = "LOW",
    INFORMATIONAL = "INFORMATIONAL"
}

/**
 * Recommendation status
 */
export enum RecommendationStatus {
    PENDING = "PENDING",
    IN_PROGRESS = "IN_PROGRESS",
    COMPLETED = "COMPLETED",
    DEFERRED = "DEFERRED",
    REJECTED = "REJECTED"
}
