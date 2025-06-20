/**
 * Enhanced system event fixtures using the factory pattern for testing
 * infrastructure monitoring, deployment processes, and incident management.
 */
import { BaseEventFactory } from "./BaseEventFactory.js";
import { type BaseEvent } from "./types.js";

// ========================================
// Core Event Type Definitions
// ========================================

interface SystemEvent<T> extends BaseEvent {
    event: string;
    data: T;
}

// ========================================
// System Health Event Types
// ========================================

interface SystemHealthData {
    status: "healthy" | "degraded" | "outage" | "recovering";
    component?: string;
    message?: string;
    timestamp: number;
    severity?: "low" | "medium" | "high" | "critical";
    services?: Record<string, "operational" | "degraded" | "down">;
    metrics?: {
        cpu: number;
        memory: number;
        activeConnections: number;
        errorRate?: number;
        responseTime?: number;
    };
    checks?: Array<{
        name: string;
        status: "pass" | "warn" | "fail";
        message?: string;
        duration?: number;
    }>;
}

export class SystemHealthEventFactory extends BaseEventFactory<SystemEvent<SystemHealthData>, SystemHealthData> {
    private healthState: SystemHealthData["status"] = "healthy";
    private componentStates: Map<string, SystemHealthData["status"]> = new Map();

    constructor() {
        super("systemHealth", {
            defaults: {
                status: "healthy",
                timestamp: Date.now(),
                severity: "low",
                services: {
                    api: "operational",
                    database: "operational",
                    cache: "operational",
                    websocket: "operational",
                },
                metrics: {
                    cpu: 25,
                    memory: 60,
                    activeConnections: 150,
                    errorRate: 0.1,
                    responseTime: 250,
                },
            },
            validation: (data: SystemHealthData) => {
                if (!["healthy", "degraded", "outage", "recovering"].includes(data.status)) {
                    return "status must be one of: healthy, degraded, outage, recovering";
                }
                if (data.severity && !["low", "medium", "high", "critical"].includes(data.severity)) {
                    return "severity must be one of: low, medium, high, critical";
                }
                if (typeof data.timestamp !== "number") {
                    return "timestamp must be a number";
                }
                return true;
            },
        });
    }

    get single(): SystemEvent<SystemHealthData> {
        return this.create();
    }

    get sequence(): SystemEvent<SystemHealthData>[] {
        return [
            this.createHealthStatus("healthy"),
            this.createHealthStatus("degraded"),
            this.createHealthStatus("outage"),
            this.createHealthStatus("recovering"),
            this.createHealthStatus("healthy"),
        ];
    }

    get variants(): Record<string, SystemEvent<SystemHealthData> | SystemEvent<SystemHealthData>[]> {
        return {
            healthy: this.createHealthStatus("healthy"),
            degraded: this.createHealthStatus("degraded"),
            outage: this.createHealthStatus("outage"),
            recovering: this.createHealthStatus("recovering"),
            componentFailure: this.createComponentFailure("database"),
            loadSpike: this.createLoadSpike(),
            degradationSequence: this.createDegradationSequence(),
            recoverySequence: this.createRecoverySequence(),
        };
    }

    createHealthStatus(status: SystemHealthData["status"], component?: string): SystemEvent<SystemHealthData> {
        this.healthState = status;
        if (component) {
            this.componentStates.set(component, status);
        }

        const statusConfig = this.getStatusConfig(status);

        return this.create({
            status,
            component,
            timestamp: Date.now(),
            ...statusConfig,
        });
    }

    createComponentFailure(component: string): SystemEvent<SystemHealthData> {
        this.componentStates.set(component, "outage");

        return this.create({
            status: "degraded",
            component,
            message: `${component} service is experiencing issues`,
            severity: "high",
            timestamp: Date.now(),
            services: {
                api: component === "api" ? "down" : "operational",
                database: component === "database" ? "down" : "operational",
                cache: component === "cache" ? "down" : "operational",
                websocket: component === "websocket" ? "down" : "operational",
            },
            checks: [{
                name: `${component}_health_check`,
                status: "fail",
                message: `${component} health check failed`,
                duration: 5000,
            }],
        });
    }

    createLoadSpike(): SystemEvent<SystemHealthData> {
        return this.create({
            status: "degraded",
            message: "High load detected across system",
            severity: "medium",
            timestamp: Date.now(),
            metrics: {
                cpu: 85,
                memory: 90,
                activeConnections: 1500,
                errorRate: 2.5,
                responseTime: 2500,
            },
        });
    }

    createDegradationSequence(): SystemEvent<SystemHealthData>[] {
        return [
            this.createHealthStatus("healthy"),
            this.createLoadSpike(),
            this.createHealthStatus("degraded"),
            this.createComponentFailure("cache"),
            this.createHealthStatus("outage"),
        ];
    }

    createRecoverySequence(): SystemEvent<SystemHealthData>[] {
        return [
            this.createHealthStatus("outage"),
            this.createHealthStatus("recovering"),
            this.createHealthStatus("degraded"),
            this.createHealthStatus("healthy"),
        ];
    }

    protected applyEventToState(state: Record<string, unknown>, event: SystemEvent<SystemHealthData>): Record<string, unknown> {
        return {
            ...state,
            systemStatus: event.data.status,
            lastHealthCheck: event.data.timestamp,
            componentStates: Object.fromEntries(this.componentStates),
        };
    }

    private getStatusConfig(status: SystemHealthData["status"]) {
        const configs = {
            healthy: {
                severity: "low" as const,
                message: "All systems operational",
                services: {
                    api: "operational" as const,
                    database: "operational" as const,
                    cache: "operational" as const,
                    websocket: "operational" as const,
                },
                metrics: {
                    cpu: 25,
                    memory: 60,
                    activeConnections: 150,
                    errorRate: 0.1,
                    responseTime: 250,
                },
            },
            degraded: {
                severity: "medium" as const,
                message: "System performance degraded",
                services: {
                    api: "operational" as const,
                    database: "operational" as const,
                    cache: "degraded" as const,
                    websocket: "operational" as const,
                },
                metrics: {
                    cpu: 75,
                    memory: 85,
                    activeConnections: 800,
                    errorRate: 2.0,
                    responseTime: 1500,
                },
            },
            outage: {
                severity: "critical" as const,
                message: "System outage detected",
                services: {
                    api: "degraded" as const,
                    database: "down" as const,
                    cache: "down" as const,
                    websocket: "degraded" as const,
                },
                metrics: {
                    cpu: 95,
                    memory: 98,
                    activeConnections: 50,
                    errorRate: 15.0,
                    responseTime: 5000,
                },
            },
            recovering: {
                severity: "medium" as const,
                message: "System recovering from outage",
                services: {
                    api: "operational" as const,
                    database: "degraded" as const,
                    cache: "operational" as const,
                    websocket: "operational" as const,
                },
                metrics: {
                    cpu: 60,
                    memory: 70,
                    activeConnections: 300,
                    errorRate: 1.0,
                    responseTime: 800,
                },
            },
        };

        return configs[status];
    }
}

// ========================================
// Maintenance Event Types
// ========================================

interface MaintenanceData {
    type: "scheduled" | "emergency" | "completed";
    title: string;
    description: string;
    startTime?: number;
    endTime?: number;
    estimatedDuration?: number;
    actualDuration?: number;
    affectedServices: string[];
    impact: "none" | "partial" | "full";
    status: "planned" | "starting" | "in_progress" | "completed" | "cancelled";
    changes?: string[];
    rollbackPlan?: string;
}

export class MaintenanceEventFactory extends BaseEventFactory<SystemEvent<MaintenanceData>, MaintenanceData> {
    private maintenanceId = 0;
    private activeMaintenances: Map<string, MaintenanceData> = new Map();

    constructor() {
        super("maintenanceNotice", {
            defaults: {
                type: "scheduled",
                title: "Scheduled Maintenance",
                description: "System maintenance window",
                affectedServices: ["api"],
                impact: "partial",
                status: "planned",
            },
            validation: (data: MaintenanceData) => {
                if (!["scheduled", "emergency", "completed"].includes(data.type)) {
                    return "type must be one of: scheduled, emergency, completed";
                }
                if (!["none", "partial", "full"].includes(data.impact)) {
                    return "impact must be one of: none, partial, full";
                }
                if (!Array.isArray(data.affectedServices)) {
                    return "affectedServices must be an array";
                }
                return true;
            },
        });
    }

    get single(): SystemEvent<MaintenanceData> {
        return this.create();
    }

    get sequence(): SystemEvent<MaintenanceData>[] {
        const maintenanceId = `maint_${++this.maintenanceId}`;
        const startTime = Date.now() + 86400000; // 24 hours from now
        const duration = 3600000; // 1 hour

        return [
            this.createScheduledMaintenance(maintenanceId, startTime, duration),
            this.createMaintenanceStarting(maintenanceId),
            this.createMaintenanceInProgress(maintenanceId),
            this.createMaintenanceCompleted(maintenanceId, duration),
        ];
    }

    get variants(): Record<string, SystemEvent<MaintenanceData> | SystemEvent<MaintenanceData>[]> {
        return {
            scheduled: this.createScheduledMaintenance(),
            emergency: this.createEmergencyMaintenance(),
            starting: this.createMaintenanceStarting(),
            inProgress: this.createMaintenanceInProgress(),
            completed: this.createMaintenanceCompleted(),
            cancelled: this.createMaintenanceCancelled(),
            fullMaintenanceWindow: this.createFullMaintenanceWindow(),
            emergencyResponse: this.createEmergencyMaintenanceSequence(),
        };
    }

    createScheduledMaintenance(
        maintenanceId?: string,
        startTime?: number,
        duration?: number,
    ): SystemEvent<MaintenanceData> {
        const id = maintenanceId || `maint_${++this.maintenanceId}`;
        const start = startTime || Date.now() + 86400000;
        const estimatedDuration = duration || 3600000;

        const data: MaintenanceData = {
            type: "scheduled",
            title: "Scheduled Maintenance Window",
            description: "Planned system upgrades and optimizations",
            startTime: start,
            endTime: start + estimatedDuration,
            estimatedDuration,
            affectedServices: ["api", "websocket"],
            impact: "partial",
            status: "planned",
            rollbackPlan: "Automatic rollback if issues detected",
        };

        this.activeMaintenances.set(id, data);
        return this.create(data);
    }

    createEmergencyMaintenance(): SystemEvent<MaintenanceData> {
        const id = `emerg_${++this.maintenanceId}`;
        const data: MaintenanceData = {
            type: "emergency",
            title: "Emergency Maintenance",
            description: "Critical security patch deployment",
            startTime: Date.now(),
            estimatedDuration: 1800000, // 30 minutes
            affectedServices: ["api", "database", "websocket"],
            impact: "full",
            status: "starting",
        };

        this.activeMaintenances.set(id, data);
        return this.create(data);
    }

    createMaintenanceStarting(maintenanceId?: string): SystemEvent<MaintenanceData> {
        const id = maintenanceId || `maint_${this.maintenanceId}`;
        const existing = this.activeMaintenances.get(id);

        const data: MaintenanceData = {
            ...existing,
            type: "scheduled",
            title: "Maintenance Starting",
            description: "Maintenance window is now beginning",
            startTime: Date.now(),
            status: "starting",
        } as MaintenanceData;

        this.activeMaintenances.set(id, data);
        return this.create(data);
    }

    createMaintenanceInProgress(maintenanceId?: string): SystemEvent<MaintenanceData> {
        const id = maintenanceId || `maint_${this.maintenanceId}`;
        const existing = this.activeMaintenances.get(id);

        const data: MaintenanceData = {
            ...existing,
            title: "Maintenance In Progress",
            description: "System maintenance is currently underway",
            status: "in_progress",
        } as MaintenanceData;

        return this.create(data);
    }

    createMaintenanceCompleted(maintenanceId?: string, actualDuration?: number): SystemEvent<MaintenanceData> {
        const id = maintenanceId || `maint_${this.maintenanceId}`;
        const existing = this.activeMaintenances.get(id);

        const data: MaintenanceData = {
            ...existing,
            type: "completed",
            title: "Maintenance Completed",
            description: "All systems have been restored",
            actualDuration: actualDuration || 3600000,
            status: "completed",
            changes: [
                "Database schema updated",
                "Security patches applied",
                "Performance optimizations implemented",
            ],
        } as MaintenanceData;

        this.activeMaintenances.delete(id);
        return this.create(data);
    }

    createMaintenanceCancelled(): SystemEvent<MaintenanceData> {
        return this.create({
            type: "scheduled",
            title: "Maintenance Cancelled",
            description: "Scheduled maintenance has been cancelled",
            affectedServices: [],
            impact: "none",
            status: "cancelled",
        });
    }

    createFullMaintenanceWindow(): SystemEvent<MaintenanceData>[] {
        const id = `maint_${++this.maintenanceId}`;
        const startTime = Date.now() + 3600000; // 1 hour from now
        const duration = 7200000; // 2 hours

        return [
            this.createScheduledMaintenance(id, startTime, duration),
            this.createMaintenanceStarting(id),
            this.createMaintenanceInProgress(id),
            this.createMaintenanceCompleted(id, duration - 600000), // Completed 10 minutes early
        ];
    }

    createEmergencyMaintenanceSequence(): SystemEvent<MaintenanceData>[] {
        return [
            this.createEmergencyMaintenance(),
            this.createMaintenanceInProgress(),
            this.createMaintenanceCompleted(),
        ];
    }

    protected applyEventToState(state: Record<string, unknown>, event: SystemEvent<MaintenanceData>): Record<string, unknown> {
        return {
            ...state,
            maintenanceStatus: event.data.status,
            lastMaintenance: event.data.type === "completed" ? Date.now() : state.lastMaintenance,
            activeMaintenance: event.data.status === "completed" ? null : event.data,
        };
    }
}

// ========================================
// Performance Event Types  
// ========================================

interface PerformanceData {
    type: "alert" | "warning" | "critical" | "recovered";
    metric: "response_time" | "throughput" | "error_rate" | "cpu_usage" | "memory_usage" | "disk_usage" | "network_latency";
    current: number;
    threshold: number;
    unit: string;
    service: string;
    timestamp: number;
    message: string;
    duration?: number;
    trend?: "increasing" | "decreasing" | "stable";
    recommendations?: string[];
    percentiles?: {
        p50: number;
        p90: number;
        p95: number;
        p99: number;
    };
}

export class PerformanceEventFactory extends BaseEventFactory<SystemEvent<PerformanceData>, PerformanceData> {
    private alertCounter = 0;
    private activeAlerts: Map<string, PerformanceData> = new Map();

    constructor() {
        super("performanceAlert", {
            defaults: {
                type: "alert",
                metric: "response_time",
                current: 1500,
                threshold: 2000,
                unit: "ms",
                service: "api",
                timestamp: Date.now(),
                message: "Performance metric alert",
                trend: "stable",
            },
            validation: (data: PerformanceData) => {
                if (!["alert", "warning", "critical", "recovered"].includes(data.type)) {
                    return "type must be one of: alert, warning, critical, recovered";
                }
                if (typeof data.current !== "number" || typeof data.threshold !== "number") {
                    return "current and threshold must be numbers";
                }
                if (!data.service || !data.metric) {
                    return "service and metric are required";
                }
                return true;
            },
        });
    }

    get single(): SystemEvent<PerformanceData> {
        return this.create();
    }

    get sequence(): SystemEvent<PerformanceData>[] {
        const alertId = `perf_${++this.alertCounter}`;
        return [
            this.createPerformanceAlert("response_time", 2500, 2000, "api"),
            this.createPerformanceWarning("response_time", 3500, 2000, "api"),
            this.createPerformanceCritical("response_time", 5000, 2000, "api"),
            this.createPerformanceRecovered("response_time", 1200, 2000, "api", 900000),
        ];
    }

    get variants(): Record<string, SystemEvent<PerformanceData> | SystemEvent<PerformanceData>[]> {
        return {
            responseTimeAlert: this.createPerformanceAlert("response_time", 2500, 2000, "api"),
            errorRateWarning: this.createPerformanceWarning("error_rate", 7.5, 5, "api"),
            cpuCritical: this.createPerformanceCritical("cpu_usage", 95, 80, "worker"),
            memoryAlert: this.createPerformanceAlert("memory_usage", 85, 80, "database"),
            throughputDrop: this.createThroughputAlert(),
            latencySpike: this.createLatencySpike(),
            escalatingAlert: this.createEscalatingAlertSequence(),
            performanceRecovery: this.createPerformanceRecoverySequence(),
        };
    }

    createPerformanceAlert(
        metric: PerformanceData["metric"],
        current: number,
        threshold: number,
        service: string,
    ): SystemEvent<PerformanceData> {
        const alertId = `perf_${++this.alertCounter}`;
        const data: PerformanceData = {
            type: "alert",
            metric,
            current,
            threshold,
            unit: this.getMetricUnit(metric),
            service,
            timestamp: Date.now(),
            message: `${service} ${metric} (${current}${this.getMetricUnit(metric)}) exceeds threshold (${threshold}${this.getMetricUnit(metric)})`,
            trend: current > threshold * 1.2 ? "increasing" : "stable",
        };

        this.activeAlerts.set(alertId, data);
        return this.create(data);
    }

    createPerformanceWarning(
        metric: PerformanceData["metric"],
        current: number,
        threshold: number,
        service: string,
    ): SystemEvent<PerformanceData> {
        const data: PerformanceData = {
            type: "warning",
            metric,
            current,
            threshold,
            unit: this.getMetricUnit(metric),
            service,
            timestamp: Date.now(),
            message: `${service} ${metric} significantly elevated`,
            trend: "increasing",
            recommendations: this.getRecommendations(metric, service),
        };

        return this.create(data);
    }

    createPerformanceCritical(
        metric: PerformanceData["metric"],
        current: number,
        threshold: number,
        service: string,
    ): SystemEvent<PerformanceData> {
        const data: PerformanceData = {
            type: "critical",
            metric,
            current,
            threshold,
            unit: this.getMetricUnit(metric),
            service,
            timestamp: Date.now(),
            message: `CRITICAL: ${service} ${metric} has reached dangerous levels`,
            trend: "increasing",
            recommendations: [
                "Immediate intervention required",
                ...this.getRecommendations(metric, service),
            ],
            percentiles: metric === "response_time" ? {
                p50: current * 0.6,
                p90: current * 0.9,
                p95: current,
                p99: current * 1.5,
            } : undefined,
        };

        return this.create(data);
    }

    createPerformanceRecovered(
        metric: PerformanceData["metric"],
        current: number,
        threshold: number,
        service: string,
        duration?: number,
    ): SystemEvent<PerformanceData> {
        const data: PerformanceData = {
            type: "recovered",
            metric,
            current,
            threshold,
            unit: this.getMetricUnit(metric),
            service,
            timestamp: Date.now(),
            message: `${service} ${metric} has returned to normal levels`,
            duration: duration || 600000,
            trend: "decreasing",
        };

        return this.create(data);
    }

    createThroughputAlert(): SystemEvent<PerformanceData> {
        return this.create({
            type: "warning",
            metric: "throughput",
            current: 50,
            threshold: 100,
            unit: "req/s",
            service: "api",
            timestamp: Date.now(),
            message: "API throughput has dropped significantly",
            trend: "decreasing",
            recommendations: [
                "Check for bottlenecks in request processing",
                "Monitor database connection pool",
                "Review recent deployments",
            ],
        });
    }

    createLatencySpike(): SystemEvent<PerformanceData> {
        return this.create({
            type: "critical",
            metric: "network_latency",
            current: 500,
            threshold: 100,
            unit: "ms",
            service: "database",
            timestamp: Date.now(),
            message: "Database latency spike detected",
            trend: "increasing",
            percentiles: {
                p50: 300,
                p90: 450,
                p95: 500,
                p99: 800,
            },
        });
    }

    createEscalatingAlertSequence(): SystemEvent<PerformanceData>[] {
        return [
            this.createPerformanceAlert("response_time", 2200, 2000, "api"),
            this.createPerformanceWarning("response_time", 3000, 2000, "api"),
            this.createPerformanceCritical("response_time", 4500, 2000, "api"),
        ];
    }

    createPerformanceRecoverySequence(): SystemEvent<PerformanceData>[] {
        return [
            this.createPerformanceCritical("cpu_usage", 95, 80, "worker"),
            this.createPerformanceWarning("cpu_usage", 85, 80, "worker"),
            this.createPerformanceAlert("cpu_usage", 82, 80, "worker"),
            this.createPerformanceRecovered("cpu_usage", 65, 80, "worker", 1200000),
        ];
    }

    protected applyEventToState(state: Record<string, unknown>, event: SystemEvent<PerformanceData>): Record<string, unknown> {
        const activeAlerts = (state.activePerformanceAlerts as string[]) || [];
        const alertKey = `${event.data.service}_${event.data.metric}`;

        return {
            ...state,
            lastPerformanceEvent: event.data.timestamp,
            activePerformanceAlerts: event.data.type === "recovered"
                ? activeAlerts.filter(alert => alert !== alertKey)
                : [...activeAlerts.filter(alert => alert !== alertKey), alertKey],
        };
    }

    private getMetricUnit(metric: PerformanceData["metric"]): string {
        const units = {
            response_time: "ms",
            throughput: "req/s",
            error_rate: "%",
            cpu_usage: "%",
            memory_usage: "%",
            disk_usage: "%",
            network_latency: "ms",
        };
        return units[metric] || "units";
    }

    private getRecommendations(metric: PerformanceData["metric"], service: string): string[] {
        const recommendations = {
            response_time: [
                "Check database query performance",
                "Review caching strategies",
                "Consider scaling horizontally",
            ],
            error_rate: [
                "Review recent deployments",
                "Check error logs for patterns",
                "Verify external service dependencies",
            ],
            cpu_usage: [
                "Scale up instance size",
                "Optimize resource-intensive operations",
                "Review background job scheduling",
            ],
            memory_usage: [
                "Check for memory leaks",
                "Optimize data structures",
                "Consider garbage collection tuning",
            ],
            throughput: [
                "Check load balancer configuration",
                "Review connection pool settings",
                "Monitor upstream dependencies",
            ],
            disk_usage: [
                "Clean up temporary files",
                "Archive old logs",
                "Increase disk capacity",
            ],
            network_latency: [
                "Check network connectivity",
                "Review DNS resolution",
                "Monitor CDN performance",
            ],
        };

        return recommendations[metric] || ["Monitor the situation closely"];
    }
}

// ========================================
// Security Event Types
// ========================================

interface SecurityData {
    type: "breach_detected" | "suspicious_activity" | "rate_limit_exceeded" | "unauthorized_access" | "malware_detected" | "ddos_attack";
    severity: "low" | "medium" | "high" | "critical";
    description: string;
    source: string;
    timestamp: number;
    details: Record<string, unknown>;
    action: string;
    status: "detected" | "investigating" | "mitigated" | "resolved";
    affectedSystems?: string[];
    impact?: "none" | "low" | "medium" | "high" | "critical";
    mitigationSteps?: string[];
    correlationId?: string;
}

export class SecurityEventFactory extends BaseEventFactory<SystemEvent<SecurityData>, SecurityData> {
    private incidentCounter = 0;
    private activeIncidents: Map<string, SecurityData> = new Map();

    constructor() {
        super("securityAlert", {
            defaults: {
                type: "suspicious_activity",
                severity: "medium",
                description: "Security event detected",
                source: "security_monitor",
                timestamp: Date.now(),
                details: {},
                action: "Monitoring situation",
                status: "detected",
                impact: "low",
            },
            validation: (data: SecurityData) => {
                if (!["breach_detected", "suspicious_activity", "rate_limit_exceeded", "unauthorized_access", "malware_detected", "ddos_attack"].includes(data.type)) {
                    return "type must be a valid security event type";
                }
                if (!["low", "medium", "high", "critical"].includes(data.severity)) {
                    return "severity must be one of: low, medium, high, critical";
                }
                if (!data.description || !data.source) {
                    return "description and source are required";
                }
                return true;
            },
        });
    }

    get single(): SystemEvent<SecurityData> {
        return this.create();
    }

    get sequence(): SystemEvent<SecurityData>[] {
        const correlationId = `sec_${++this.incidentCounter}`;
        return [
            this.createSuspiciousActivity(correlationId),
            this.createSecurityBreach(correlationId),
            this.createSecurityMitigation(correlationId),
            this.createSecurityResolved(correlationId),
        ];
    }

    get variants(): Record<string, SystemEvent<SecurityData> | SystemEvent<SecurityData>[]> {
        return {
            suspiciousActivity: this.createSuspiciousActivity(),
            breachDetected: this.createSecurityBreach(),
            rateLimitExceeded: this.createRateLimitExceeded(),
            unauthorizedAccess: this.createUnauthorizedAccess(),
            malwareDetected: this.createMalwareDetection(),
            ddosAttack: this.createDDoSAttack(),
            incidentResponse: this.createIncidentResponseSequence(),
            falsePositive: this.createFalsePositiveSequence(),
        };
    }

    createSuspiciousActivity(correlationId?: string): SystemEvent<SecurityData> {
        const id = correlationId || `sec_${++this.incidentCounter}`;
        const data: SecurityData = {
            type: "suspicious_activity",
            severity: "medium",
            description: "Multiple failed authentication attempts detected",
            source: "auth_service",
            timestamp: Date.now(),
            details: {
                ip: "203.0.113.45",
                attempts: 15,
                timeframe: 300000, // 5 minutes
                userAgent: "Mozilla/5.0 (suspicious pattern)",
                endpoints: ["/api/auth/login", "/api/auth/reset"],
            },
            action: "IP temporarily blocked for 1 hour",
            status: "detected",
            impact: "low",
            correlationId: id,
        };

        this.activeIncidents.set(id, data);
        return this.create(data);
    }

    createSecurityBreach(correlationId?: string): SystemEvent<SecurityData> {
        const id = correlationId || `sec_${++this.incidentCounter}`;
        const data: SecurityData = {
            type: "breach_detected",
            severity: "critical",
            description: "Potential data breach detected - unauthorized database access",
            source: "database_monitor",
            timestamp: Date.now(),
            details: {
                database: "user_data",
                query: "SELECT * FROM users WHERE admin=true",
                source_ip: "203.0.113.45",
                affected_records: 0, // Blocked before execution
                detection_method: "anomaly_detection",
            },
            action: "Database access blocked, incident escalated",
            status: "investigating",
            impact: "high",
            affectedSystems: ["user_database", "authentication_service"],
            mitigationSteps: [
                "Block suspicious IP address",
                "Revoke all active sessions for affected accounts",
                "Enable enhanced monitoring",
                "Initiate security audit",
            ],
            correlationId: id,
        };

        this.activeIncidents.set(id, data);
        return this.create(data);
    }

    createRateLimitExceeded(): SystemEvent<SecurityData> {
        return this.create({
            type: "rate_limit_exceeded",
            severity: "low",
            description: "API rate limit exceeded by client",
            source: "api_gateway",
            timestamp: Date.now(),
            details: {
                client_id: "app_12345",
                endpoint: "/api/data/export",
                requests_per_minute: 150,
                limit: 100,
                burst_requests: 75,
            },
            action: "Rate limiting enforced, requests throttled",
            status: "mitigated",
            impact: "none",
        });
    }

    createUnauthorizedAccess(): SystemEvent<SecurityData> {
        return this.create({
            type: "unauthorized_access",
            severity: "high",
            description: "Unauthorized access attempt to admin panel",
            source: "access_control",
            timestamp: Date.now(),
            details: {
                user_id: "user_suspicious_789",
                attempted_resource: "/admin/system/config",
                access_level: "standard_user",
                required_level: "admin",
                session_id: "sess_abc123def456",
            },
            action: "Access denied, user flagged for review",
            status: "detected",
            impact: "medium",
            affectedSystems: ["admin_panel", "access_control"],
        });
    }

    createMalwareDetection(): SystemEvent<SecurityData> {
        return this.create({
            type: "malware_detected",
            severity: "critical",
            description: "Malicious file upload detected and quarantined",
            source: "file_scanner",
            timestamp: Date.now(),
            details: {
                filename: "document.pdf.exe",
                file_hash: "a1b2c3d4e5f6789abcdef123456789",
                malware_type: "trojan.generic",
                upload_source: "user_content",
                quarantine_location: "/quarantine/2024/01/suspicious_file_001",
            },
            action: "File quarantined, upload blocked, user notified",
            status: "mitigated",
            impact: "none",
            mitigationSteps: [
                "File moved to quarantine",
                "User account flagged for review",
                "Enhanced scanning enabled",
            ],
        });
    }

    createDDoSAttack(): SystemEvent<SecurityData> {
        return this.create({
            type: "ddos_attack",
            severity: "critical",
            description: "Distributed Denial of Service attack in progress",
            source: "network_monitor",
            timestamp: Date.now(),
            details: {
                attack_type: "volumetric",
                peak_traffic: "10.5 Gbps",
                source_ips: 450,
                target_endpoints: ["/api/public", "/health", "/status"],
                duration: 1800000, // 30 minutes so far
                geographic_distribution: ["CN", "RU", "BR", "IN"],
            },
            action: "DDoS protection activated, traffic filtered",
            status: "mitigated",
            impact: "medium",
            affectedSystems: ["api_gateway", "load_balancer"],
            mitigationSteps: [
                "Activate DDoS protection",
                "Enable geo-blocking for suspicious regions",
                "Scale infrastructure to handle load",
                "Contact ISP for upstream filtering",
            ],
        });
    }

    createSecurityMitigation(correlationId: string): SystemEvent<SecurityData> {
        const existing = this.activeIncidents.get(correlationId);
        if (!existing) {
            throw new Error(`No active incident found for correlation ID: ${correlationId}`);
        }

        const data: SecurityData = {
            ...existing,
            status: "mitigated",
            action: "Security measures implemented successfully",
            timestamp: Date.now(),
        };

        this.activeIncidents.set(correlationId, data);
        return this.create(data);
    }

    createSecurityResolved(correlationId: string): SystemEvent<SecurityData> {
        const existing = this.activeIncidents.get(correlationId);
        if (!existing) {
            throw new Error(`No active incident found for correlation ID: ${correlationId}`);
        }

        const data: SecurityData = {
            ...existing,
            status: "resolved",
            action: "Incident resolved, normal operations restored",
            timestamp: Date.now(),
            impact: "none",
        };

        this.activeIncidents.delete(correlationId);
        return this.create(data);
    }

    createIncidentResponseSequence(): SystemEvent<SecurityData>[] {
        const correlationId = `sec_${++this.incidentCounter}`;
        return [
            this.createSuspiciousActivity(correlationId),
            this.createSecurityBreach(correlationId),
            this.createSecurityMitigation(correlationId),
            this.createSecurityResolved(correlationId),
        ];
    }

    createFalsePositiveSequence(): SystemEvent<SecurityData>[] {
        const correlationId = `sec_${++this.incidentCounter}`;
        return [
            this.createSuspiciousActivity(correlationId),
            this.create({
                type: "suspicious_activity",
                severity: "low",
                description: "False positive - legitimate user behavior",
                source: "security_analyst",
                timestamp: Date.now(),
                details: {
                    analysis: "Manual review confirmed legitimate activity",
                    user_verification: "passed",
                    behavior_pattern: "normal_for_user",
                },
                action: "Alert dismissed, user whitelist updated",
                status: "resolved",
                impact: "none",
                correlationId,
            }),
        ];
    }

    protected applyEventToState(state: Record<string, unknown>, event: SystemEvent<SecurityData>): Record<string, unknown> {
        const activeIncidents = (state.activeSecurityIncidents as string[]) || [];
        const incidentId = event.data.correlationId || `${event.data.source}_${event.data.timestamp}`;

        return {
            ...state,
            securityLevel: this.calculateSecurityLevel(event.data.severity, activeIncidents.length),
            lastSecurityEvent: event.data.timestamp,
            activeSecurityIncidents: event.data.status === "resolved"
                ? activeIncidents.filter(id => id !== incidentId)
                : [...activeIncidents.filter(id => id !== incidentId), incidentId],
        };
    }

    private calculateSecurityLevel(severity: SecurityData["severity"], activeIncidents: number): string {
        if (severity === "critical" || activeIncidents > 5) return "high_alert";
        if (severity === "high" || activeIncidents > 2) return "elevated";
        if (severity === "medium" || activeIncidents > 0) return "guarded";
        return "normal";
    }
}

// ========================================
// Exported Factory Instances and Legacy Support
// ========================================

export const systemHealthEventFactory = new SystemHealthEventFactory();
export const maintenanceEventFactory = new MaintenanceEventFactory();
export const performanceEventFactory = new PerformanceEventFactory();
export const securityEventFactory = new SecurityEventFactory();

// Legacy systemEventFixtures for backward compatibility
export const systemEventFixtures = {
    status: {
        healthy: systemHealthEventFactory.variants.healthy as SystemEvent<SystemHealthData>,
        degraded: systemHealthEventFactory.variants.degraded as SystemEvent<SystemHealthData>,
        critical: systemHealthEventFactory.variants.outage as SystemEvent<SystemHealthData>,
    },
    maintenance: {
        scheduled: maintenanceEventFactory.variants.scheduled as SystemEvent<MaintenanceData>,
        starting: maintenanceEventFactory.variants.starting as SystemEvent<MaintenanceData>,
        complete: maintenanceEventFactory.variants.completed as SystemEvent<MaintenanceData>,
    },
    deployment: {
        started: maintenanceEventFactory.create({
            type: "scheduled",
            title: "Deployment Started",
            description: "Application deployment in progress",
            affectedServices: ["api", "worker"],
            impact: "partial",
            status: "starting",
        }),
        progress: maintenanceEventFactory.variants.inProgress as SystemEvent<MaintenanceData>,
        completed: maintenanceEventFactory.variants.completed as SystemEvent<MaintenanceData>,
        failed: maintenanceEventFactory.create({
            type: "emergency",
            title: "Deployment Failed",
            description: "Deployment failed, initiating rollback",
            affectedServices: ["api"],
            impact: "full",
            status: "cancelled",
        }),
        rolledBack: maintenanceEventFactory.create({
            type: "completed",
            title: "Deployment Rolled Back",
            description: "System restored to previous version",
            affectedServices: ["api"],
            impact: "none",
            status: "completed",
        }),
    },
    performance: {
        alert: performanceEventFactory.variants.responseTimeAlert as SystemEvent<PerformanceData>,
        critical: performanceEventFactory.variants.errorRateWarning as SystemEvent<PerformanceData>,
        recovered: performanceEventFactory.createPerformanceRecovered("response_time", 1200, 2000, "api"),
    },
    security: {
        alert: securityEventFactory.variants.suspiciousActivity as SystemEvent<SecurityData>,
        incident: securityEventFactory.variants.breachDetected as SystemEvent<SecurityData>,
        update: securityEventFactory.create({
            type: "breach_detected",
            severity: "critical",
            description: "Security update applied",
            source: "security_team",
            timestamp: Date.now(),
            details: {
                update_type: "security_patch",
                cve: "CVE-2024-12345",
                components: ["authentication", "api"],
            },
            action: "Critical security patch applied",
            status: "resolved",
            impact: "none",
        }),
    },
    sequences: {
        healthDegradationFlow: systemHealthEventFactory.variants.degradationSequence as SystemEvent<SystemHealthData>[],
        deploymentFlow: maintenanceEventFactory.variants.fullMaintenanceWindow as SystemEvent<MaintenanceData>[],
        failedDeploymentFlow: maintenanceEventFactory.variants.emergencyResponse as SystemEvent<MaintenanceData>[],
        maintenanceFlow: maintenanceEventFactory.variants.fullMaintenanceWindow as SystemEvent<MaintenanceData>[],
        securityIncidentFlow: securityEventFactory.variants.incidentResponse as SystemEvent<SecurityData>[],
    },
    factories: {
        createSystemStatus: (status: "healthy" | "degraded" | "critical", services?: Record<string, string>) =>
            systemHealthEventFactory.createHealthStatus(
                status === "critical" ? "outage" : status,
                services ? Object.keys(services)[0] : undefined,
            ),
        createPerformanceAlert: (metric: string, current: number, threshold: number, service: string) =>
            performanceEventFactory.createPerformanceAlert(metric as PerformanceData["metric"], current, threshold, service),
        createSecurityAlert: (type: string, severity: "low" | "medium" | "high" | "critical", description: string) =>
            securityEventFactory.create({
                type: type as SecurityData["type"],
                severity,
                description,
                source: "security_monitor",
                timestamp: Date.now(),
                details: {},
                action: "Security alert triggered",
                status: "detected",
            }),
        createDeploymentStatus: (deploymentId: string, status: string, version: string) =>
            maintenanceEventFactory.create({
                type: "scheduled",
                title: `Deployment ${status}`,
                description: `Version ${version} deployment ${status}`,
                affectedServices: ["api"],
                impact: "partial",
                status: status as MaintenanceData["status"],
            }),
    },
};
