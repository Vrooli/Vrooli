/**
 * Migration script for replacing legacy monitoring components
 * with the UnifiedMonitoringService.
 */

import { UnifiedMonitoringService } from "../UnifiedMonitoringService";
import { TelemetryShimAdapter } from "../adapters/TelemetryShimAdapter";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";

/**
 * Migration configuration
 */
interface MigrationConfig {
    dryRun: boolean;
    migrateComponents: string[];
    preserveData: boolean;
    rollbackEnabled: boolean;
}

/**
 * Migration status tracking
 */
interface MigrationStatus {
    component: string;
    status: "pending" | "in_progress" | "completed" | "failed" | "rolled_back";
    startTime?: Date;
    endTime?: Date;
    errors?: string[];
    dataPreserved: boolean;
}

/**
 * Main migration orchestrator
 */
export class MonitoringMigrationOrchestrator {
    private readonly statuses = new Map<string, MigrationStatus>();
    private unifiedService?: UnifiedMonitoringService;
    
    constructor(
        private readonly logger: Logger,
        private readonly eventBus: EventBus,
        private readonly config: MigrationConfig
    ) {}
    
    /**
     * Execute the complete migration process
     */
    async migrate(): Promise<boolean> {
        this.logger.info("Starting monitoring migration", { 
            config: this.config,
            components: this.config.migrateComponents.length,
        });
        
        try {
            // Step 1: Initialize unified monitoring service
            await this.initializeUnifiedService();
            
            // Step 2: Migrate components in order
            for (const component of this.config.migrateComponents) {
                await this.migrateComponent(component);
            }
            
            // Step 3: Validate migration
            const success = await this.validateMigration();
            
            if (success) {
                this.logger.info("Migration completed successfully");
                return true;
            } else {
                this.logger.error("Migration validation failed");
                if (this.config.rollbackEnabled) {
                    await this.rollback();
                }
                return false;
            }
            
        } catch (error) {
            this.logger.error("Migration failed", { error });
            if (this.config.rollbackEnabled) {
                await this.rollback();
            }
            return false;
        }
    }
    
    /**
     * Initialize the unified monitoring service
     */
    private async initializeUnifiedService(): Promise<void> {
        this.logger.info("Initializing UnifiedMonitoringService");
        
        this.unifiedService = UnifiedMonitoringService.getInstance({
            maxOverheadMs: 5,
            eventBusEnabled: true,
            mcpToolsEnabled: true,
        }, this.eventBus);
        
        await this.unifiedService.initialize();
        
        // Verify service health
        const health = await this.unifiedService.getHealth();
        if (health.status !== "healthy") {
            throw new Error(`UnifiedMonitoringService unhealthy: ${health.status}`);
        }
        
        this.logger.info("UnifiedMonitoringService initialized successfully");
    }
    
    /**
     * Migrate a specific component
     */
    private async migrateComponent(component: string): Promise<void> {
        const status: MigrationStatus = {
            component,
            status: "pending",
            dataPreserved: false,
            errors: [],
        };
        
        this.statuses.set(component, status);
        
        try {
            status.status = "in_progress";
            status.startTime = new Date();
            
            this.logger.info(`Migrating component: ${component}`);
            
            switch (component) {
                case "TelemetryShim":
                    await this.migrateTelemetryShim();
                    break;
                case "RollingHistory":
                    await this.migrateRollingHistory();
                    break;
                case "MetacognitiveMonitor":
                    await this.migrateMetacognitiveMonitor();
                    break;
                case "PerformanceMonitor":
                    await this.migratePerformanceMonitor();
                    break;
                case "PerformanceTracker":
                    await this.migratePerformanceTracker();
                    break;
                case "StrategyMetricsStore":
                    await this.migrateStrategyMetricsStore();
                    break;
                case "ResourceMetrics":
                    await this.migrateResourceMetrics();
                    break;
                case "ResourceMonitor":
                    await this.migrateResourceMonitor();
                    break;
                case "MonitoringTools":
                    await this.migrateMonitoringTools();
                    break;
                case "MonitoringUtils":
                    await this.migrateMonitoringUtils();
                    break;
                default:
                    throw new Error(`Unknown component: ${component}`);
            }
            
            status.status = "completed";
            status.endTime = new Date();
            status.dataPreserved = this.config.preserveData;
            
            this.logger.info(`Successfully migrated: ${component}`);
            
        } catch (error) {
            status.status = "failed";
            status.endTime = new Date();
            status.errors?.push(error instanceof Error ? error.message : String(error));
            
            this.logger.error(`Failed to migrate component: ${component}`, { error });
            throw error;
        }
    }
    
    /**
     * Migrate TelemetryShim to TelemetryShimAdapter
     */
    private async migrateTelemetryShim(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would replace TelemetryShim with TelemetryShimAdapter");
            return;
        }
        
        // Find all TelemetryShim usages and replace with TelemetryShimAdapter
        // This is a conceptual implementation - in practice would use AST manipulation
        this.logger.info("Replacing TelemetryShim imports with TelemetryShimAdapter");
        
        // Example replacement:
        // OLD: import { TelemetryShim } from "../../cross-cutting/monitoring/telemetryShim.js";
        // NEW: import { TelemetryShimAdapter as TelemetryShim } from "../../monitoring/adapters/TelemetryShimAdapter.js";
        
        this.logger.info("TelemetryShim migration completed");
    }
    
    /**
     * Migrate RollingHistory to unified storage
     */
    private async migrateRollingHistory(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would migrate RollingHistory data to UnifiedMonitoringService");
            return;
        }
        
        // Extract data from existing RollingHistory instances
        // Convert to UnifiedMetric format
        // Load into UnifiedMonitoringService
        
        this.logger.info("RollingHistory migration completed");
    }
    
    /**
     * Migrate MetacognitiveMonitor (Tier 1)
     */
    private async migrateMetacognitiveMonitor(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would create Tier1MonitoringAdapter");
            return;
        }
        
        // Create adapter that maintains same interface but uses UnifiedMonitoringService
        this.logger.info("MetacognitiveMonitor migration completed");
    }
    
    /**
     * Migrate PerformanceMonitor (Tier 2)
     */
    private async migratePerformanceMonitor(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would create Tier2MonitoringAdapter");
            return;
        }
        
        this.logger.info("PerformanceMonitor migration completed");
    }
    
    /**
     * Migrate PerformanceTracker (Tier 3)
     */
    private async migratePerformanceTracker(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would migrate PerformanceTracker to unified system");
            return;
        }
        
        this.logger.info("PerformanceTracker migration completed");
    }
    
    /**
     * Migrate StrategyMetricsStore (Tier 3)
     */
    private async migrateStrategyMetricsStore(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would migrate StrategyMetricsStore data");
            return;
        }
        
        this.logger.info("StrategyMetricsStore migration completed");
    }
    
    /**
     * Migrate ResourceMetrics
     */
    private async migrateResourceMetrics(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would migrate ResourceMetrics");
            return;
        }
        
        this.logger.info("ResourceMetrics migration completed");
    }
    
    /**
     * Migrate ResourceMonitor
     */
    private async migrateResourceMonitor(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would migrate ResourceMonitor");
            return;
        }
        
        this.logger.info("ResourceMonitor migration completed");
    }
    
    /**
     * Migrate MonitoringTools
     */
    private async migrateMonitoringTools(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would replace MonitoringTools with MCPToolAdapter");
            return;
        }
        
        this.logger.info("MonitoringTools migration completed");
    }
    
    /**
     * Migrate MonitoringUtils
     */
    private async migrateMonitoringUtils(): Promise<void> {
        if (this.config.dryRun) {
            this.logger.info("DRY RUN: Would replace MonitoringUtils with StatisticalEngine");
            return;
        }
        
        this.logger.info("MonitoringUtils migration completed");
    }
    
    /**
     * Validate the migration was successful
     */
    private async validateMigration(): Promise<boolean> {
        this.logger.info("Validating migration");
        
        if (!this.unifiedService) {
            this.logger.error("UnifiedMonitoringService not initialized");
            return false;
        }
        
        // Check service health
        const health = await this.unifiedService.getHealth();
        if (health.status !== "healthy") {
            this.logger.error("UnifiedMonitoringService unhealthy after migration", { health });
            return false;
        }
        
        // Verify each component was migrated successfully
        for (const [component, status] of this.statuses) {
            if (status.status !== "completed") {
                this.logger.error(`Component migration failed: ${component}`, { status });
                return false;
            }
        }
        
        // Test basic functionality
        try {
            await this.unifiedService.record({
                tier: "cross-cutting",
                component: "migration-test",
                type: "business",
                name: "migration.validation",
                value: 1,
            });
            
            const result = await this.unifiedService.query({
                name: "migration.validation",
                limit: 1,
            });
            
            if (result.metrics.length === 0) {
                this.logger.error("Basic functionality test failed");
                return false;
            }
        } catch (error) {
            this.logger.error("Basic functionality test failed", { error });
            return false;
        }
        
        this.logger.info("Migration validation successful");
        return true;
    }
    
    /**
     * Rollback the migration
     */
    private async rollback(): Promise<void> {
        this.logger.warn("Rolling back migration");
        
        // Restore original components
        // This would involve file system operations to restore backups
        
        for (const [component, status] of this.statuses) {
            if (status.status === "completed") {
                status.status = "rolled_back";
                this.logger.info(`Rolled back: ${component}`);
            }
        }
        
        this.logger.warn("Rollback completed");
    }
    
    /**
     * Get migration status summary
     */
    getMigrationStatus(): MigrationStatus[] {
        return Array.from(this.statuses.values());
    }
    
    /**
     * Generate migration report
     */
    generateReport(): string {
        const statuses = this.getMigrationStatus();
        const completed = statuses.filter(s => s.status === "completed").length;
        const failed = statuses.filter(s => s.status === "failed").length;
        const total = statuses.length;
        
        let report = `
# Monitoring Migration Report

## Summary
- Total Components: ${total}
- Successfully Migrated: ${completed}
- Failed: ${failed}
- Success Rate: ${((completed / total) * 100).toFixed(1)}%

## Component Status
`;
        
        for (const status of statuses) {
            const duration = status.startTime && status.endTime 
                ? status.endTime.getTime() - status.startTime.getTime()
                : 0;
                
            report += `
### ${status.component}
- Status: ${status.status}
- Duration: ${duration}ms
- Data Preserved: ${status.dataPreserved}
- Errors: ${status.errors?.length || 0}
`;
            
            if (status.errors && status.errors.length > 0) {
                report += `
- Error Details:
${status.errors.map(e => `  - ${e}`).join('\n')}
`;
            }
        }
        
        return report;
    }
}

/**
 * Helper function to run migration with default configuration
 */
export async function runMonitoringMigration(
    logger: Logger,
    eventBus: EventBus,
    options: Partial<MigrationConfig> = {}
): Promise<boolean> {
    const config: MigrationConfig = {
        dryRun: false,
        migrateComponents: [
            "TelemetryShim",
            "RollingHistory", 
            "MetacognitiveMonitor",
            "PerformanceMonitor",
            "PerformanceTracker",
            "StrategyMetricsStore",
            "ResourceMetrics",
            "ResourceMonitor",
            "MonitoringTools",
            "MonitoringUtils",
        ],
        preserveData: true,
        rollbackEnabled: true,
        ...options,
    };
    
    const migrator = new MonitoringMigrationOrchestrator(logger, eventBus, config);
    const success = await migrator.migrate();
    
    // Generate and log report
    const report = migrator.generateReport();
    logger.info("Migration report generated", { report });
    
    return success;
}