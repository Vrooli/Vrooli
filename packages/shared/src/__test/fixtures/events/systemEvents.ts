/**
 * System-level event fixtures for testing infrastructure and status updates
 */

export const systemEventFixtures = {
    status: {
        // System healthy
        healthy: {
            event: "systemStatus",
            data: {
                status: "healthy",
                timestamp: new Date().toISOString(),
                services: {
                    api: "operational",
                    database: "operational",
                    cache: "operational",
                    queue: "operational",
                    websocket: "operational",
                },
                metrics: {
                    cpu: 25,
                    memory: 60,
                    activeConnections: 150,
                    queueDepth: 5,
                },
            },
        },

        // System degraded
        degraded: {
            event: "systemStatus",
            data: {
                status: "degraded",
                timestamp: new Date().toISOString(),
                services: {
                    api: "operational",
                    database: "operational",
                    cache: "degraded",
                    queue: "operational",
                    websocket: "operational",
                },
                metrics: {
                    cpu: 75,
                    memory: 85,
                    activeConnections: 500,
                    queueDepth: 100,
                },
                issues: ["Cache response time elevated"],
            },
        },

        // System critical
        critical: {
            event: "systemStatus",
            data: {
                status: "critical",
                timestamp: new Date().toISOString(),
                services: {
                    api: "operational",
                    database: "down",
                    cache: "down",
                    queue: "degraded",
                    websocket: "operational",
                },
                metrics: {
                    cpu: 95,
                    memory: 98,
                    activeConnections: 50,
                    queueDepth: 1000,
                },
                issues: [
                    "Database connection lost",
                    "Cache server unreachable",
                    "Queue backlog critical",
                ],
            },
        },
    },

    maintenance: {
        // Maintenance scheduled
        scheduled: {
            event: "maintenanceNotice",
            data: {
                type: "scheduled",
                title: "Scheduled Maintenance",
                description: "System will be unavailable for upgrades",
                startTime: new Date(Date.now() + 86400000).toISOString(), // 24 hours from now
                endTime: new Date(Date.now() + 90000000).toISOString(), // 25 hours from now
                affectedServices: ["api", "websocket"],
                impact: "full",
            },
        },

        // Maintenance starting
        starting: {
            event: "maintenanceNotice",
            data: {
                type: "starting",
                title: "Maintenance Starting",
                description: "System entering maintenance mode",
                startTime: new Date().toISOString(),
                estimatedDuration: 3600000, // 1 hour
            },
        },

        // Maintenance complete
        complete: {
            event: "maintenanceNotice",
            data: {
                type: "complete",
                title: "Maintenance Complete",
                description: "All services have been restored",
                completedAt: new Date().toISOString(),
                changes: [
                    "Database upgraded to v2.0",
                    "Security patches applied",
                    "Performance improvements",
                ],
            },
        },
    },

    deployment: {
        // New deployment
        started: {
            event: "deploymentStatus",
            data: {
                deploymentId: "deploy_123",
                status: "started",
                version: "v1.2.3",
                environment: "production",
                timestamp: new Date().toISOString(),
                components: ["api", "worker", "websocket"],
            },
        },

        // Deployment progress
        progress: {
            event: "deploymentStatus",
            data: {
                deploymentId: "deploy_123",
                status: "in_progress",
                version: "v1.2.3",
                environment: "production",
                timestamp: new Date().toISOString(),
                progress: {
                    api: "completed",
                    worker: "in_progress",
                    websocket: "pending",
                },
                percentage: 50,
            },
        },

        // Deployment complete
        completed: {
            event: "deploymentStatus",
            data: {
                deploymentId: "deploy_123",
                status: "completed",
                version: "v1.2.3",
                environment: "production",
                timestamp: new Date().toISOString(),
                duration: 180000, // 3 minutes
                success: true,
            },
        },

        // Deployment failed
        failed: {
            event: "deploymentStatus",
            data: {
                deploymentId: "deploy_123",
                status: "failed",
                version: "v1.2.3",
                environment: "production",
                timestamp: new Date().toISOString(),
                error: "Health check failed on api component",
                rollbackInitiated: true,
            },
        },

        // Deployment rolled back
        rolledBack: {
            event: "deploymentStatus",
            data: {
                deploymentId: "deploy_123",
                status: "rolled_back",
                version: "v1.2.3",
                previousVersion: "v1.2.2",
                environment: "production",
                timestamp: new Date().toISOString(),
                reason: "Automated rollback due to errors",
            },
        },
    },

    performance: {
        // Performance alert
        alert: {
            event: "performanceAlert",
            data: {
                type: "warning",
                metric: "response_time",
                current: 2500,
                threshold: 2000,
                unit: "ms",
                service: "api",
                timestamp: new Date().toISOString(),
                message: "API response time exceeds threshold",
            },
        },

        // Performance critical
        critical: {
            event: "performanceAlert",
            data: {
                type: "critical",
                metric: "error_rate",
                current: 15.5,
                threshold: 5,
                unit: "%",
                service: "api",
                timestamp: new Date().toISOString(),
                message: "API error rate critically high",
                recommendations: [
                    "Scale up API instances",
                    "Check database connections",
                    "Review recent deployments",
                ],
            },
        },

        // Performance recovered
        recovered: {
            event: "performanceAlert",
            data: {
                type: "recovered",
                metric: "response_time",
                current: 1200,
                threshold: 2000,
                unit: "ms",
                service: "api",
                timestamp: new Date().toISOString(),
                message: "API response time back to normal",
                duration: 900000, // Alert lasted 15 minutes
            },
        },
    },

    security: {
        // Security alert
        alert: {
            event: "securityAlert",
            data: {
                type: "suspicious_activity",
                severity: "medium",
                description: "Multiple failed login attempts detected",
                source: "auth_service",
                timestamp: new Date().toISOString(),
                details: {
                    ip: "192.168.1.100",
                    attempts: 10,
                    timeframe: 300000, // 5 minutes
                },
                action: "IP temporarily blocked",
            },
        },

        // Security incident
        incident: {
            event: "securityAlert",
            data: {
                type: "breach_attempt",
                severity: "high",
                description: "SQL injection attempt detected",
                source: "api_gateway",
                timestamp: new Date().toISOString(),
                details: {
                    endpoint: "/api/users",
                    payload: "'; DROP TABLE users; --",
                    blocked: true,
                },
                action: "Request blocked, user banned",
            },
        },

        // Security update
        update: {
            event: "securityUpdate",
            data: {
                type: "patch_applied",
                severity: "critical",
                description: "Critical security patch applied",
                timestamp: new Date().toISOString(),
                cve: "CVE-2024-12345",
                affectedComponents: ["authentication", "api"],
                status: "protected",
            },
        },
    },

    // Event sequences for testing system flows
    sequences: {
        // Health degradation flow
        healthDegradationFlow: [
            { event: "systemStatus", data: systemEventFixtures.status.healthy.data },
            { delay: 60000 },
            { event: "performanceAlert", data: systemEventFixtures.performance.alert.data },
            { delay: 30000 },
            { event: "systemStatus", data: systemEventFixtures.status.degraded.data },
            { delay: 60000 },
            { event: "performanceAlert", data: systemEventFixtures.performance.critical.data },
            { event: "systemStatus", data: systemEventFixtures.status.critical.data },
        ],

        // Deployment flow
        deploymentFlow: [
            { event: "deploymentStatus", data: systemEventFixtures.deployment.started.data },
            { delay: 30000 },
            { event: "deploymentStatus", data: systemEventFixtures.deployment.progress.data },
            { delay: 60000 },
            { event: "deploymentStatus", data: { ...systemEventFixtures.deployment.progress.data, percentage: 75 } },
            { delay: 60000 },
            { event: "deploymentStatus", data: systemEventFixtures.deployment.completed.data },
            { event: "systemStatus", data: systemEventFixtures.status.healthy.data },
        ],

        // Failed deployment with rollback
        failedDeploymentFlow: [
            { event: "deploymentStatus", data: systemEventFixtures.deployment.started.data },
            { delay: 30000 },
            { event: "deploymentStatus", data: systemEventFixtures.deployment.progress.data },
            { delay: 60000 },
            { event: "performanceAlert", data: systemEventFixtures.performance.critical.data },
            { event: "deploymentStatus", data: systemEventFixtures.deployment.failed.data },
            { delay: 10000 },
            { event: "deploymentStatus", data: systemEventFixtures.deployment.rolledBack.data },
            { event: "systemStatus", data: systemEventFixtures.status.healthy.data },
        ],

        // Maintenance window
        maintenanceFlow: [
            { event: "maintenanceNotice", data: systemEventFixtures.maintenance.scheduled.data },
            { delay: 86340000 }, // ~24 hours minus 1 minute
            { event: "maintenanceNotice", data: { ...systemEventFixtures.maintenance.scheduled.data, description: "Maintenance starting in 1 minute" } },
            { delay: 60000 },
            { event: "maintenanceNotice", data: systemEventFixtures.maintenance.starting.data },
            { event: "systemStatus", data: { ...systemEventFixtures.status.critical.data, maintenance: true } },
            { delay: 3600000 }, // 1 hour
            { event: "maintenanceNotice", data: systemEventFixtures.maintenance.complete.data },
            { event: "systemStatus", data: systemEventFixtures.status.healthy.data },
        ],

        // Security incident response
        securityIncidentFlow: [
            { event: "securityAlert", data: systemEventFixtures.security.alert.data },
            { delay: 5000 },
            { event: "securityAlert", data: { ...systemEventFixtures.security.alert.data, details: { ...systemEventFixtures.security.alert.data.details, attempts: 20 } } },
            { delay: 5000 },
            { event: "securityAlert", data: systemEventFixtures.security.incident.data },
            { event: "systemStatus", data: { ...systemEventFixtures.status.degraded.data, securityMode: "heightened" } },
            { delay: 300000 }, // 5 minutes
            { event: "securityUpdate", data: systemEventFixtures.security.update.data },
            { event: "systemStatus", data: systemEventFixtures.status.healthy.data },
        ],
    },

    // Factory functions for dynamic event creation
    factories: {
        createSystemStatus: (status: "healthy" | "degraded" | "critical", services?: Record<string, string>) => ({
            event: "systemStatus",
            data: {
                status,
                timestamp: new Date().toISOString(),
                services: services || systemEventFixtures.status[status].data.services,
                metrics: systemEventFixtures.status[status].data.metrics,
            },
        }),

        createPerformanceAlert: (metric: string, current: number, threshold: number, service: string) => ({
            event: "performanceAlert",
            data: {
                type: current > threshold ? "warning" : "recovered",
                metric,
                current,
                threshold,
                unit: metric.includes("time") ? "ms" : metric.includes("rate") ? "%" : "count",
                service,
                timestamp: new Date().toISOString(),
                message: `${service} ${metric} is ${current > threshold ? "above" : "below"} threshold`,
            },
        }),

        createSecurityAlert: (type: string, severity: "low" | "medium" | "high" | "critical", description: string) => ({
            event: "securityAlert",
            data: {
                type,
                severity,
                description,
                source: "security_monitor",
                timestamp: new Date().toISOString(),
                action: severity === "critical" ? "Immediate action required" : "Automated response initiated",
            },
        }),

        createDeploymentStatus: (deploymentId: string, status: string, version: string) => ({
            event: "deploymentStatus",
            data: {
                deploymentId,
                status,
                version,
                environment: "production",
                timestamp: new Date().toISOString(),
            },
        }),
    },
};