// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with proper types
import { generatePK, generatePublicId } from "@vrooli/shared";
import { type PrismaClient } from "@prisma/client";
import type { 
    DbTestFixtures, 
    DbFactoryConfig, 
    DbFactoryResult, 
    DbFixtureValidation,
    BulkSeedOptions,
    BulkSeedResult,
} from "./types.js";

// Relationship configuration types
export interface RelationConfig {
    [key: string]: any;
}

export interface RelationConnections {
    [key: string]: { id: string } | Array<{ id: string }>;
}

export interface RelationCounts {
    [key: string]: number;
}

export interface TestScenario {
    name: string;
    description?: string;
    config: Record<string, any>;
}

export interface ScenarioResult {
    id: string;
    scenario: string;
    relatedIds: Record<string, string[]>;
}

export interface ConstraintValidation {
    valid: boolean;
    violations: string[];
}

/**
 * Enhanced database fixture factory that implements the ideal architecture
 * This factory provides comprehensive database testing capabilities including:
 * - Relationship management
 * - Bulk operations
 * - State verification
 * - Cleanup utilities
 * - Transaction support
 */
export abstract class DatabaseFixtureFactory<
    TPrismaModel,
    TPrismaCreateInput,
    TPrismaInclude = any,
    TPrismaUpdateInput = Partial<TPrismaCreateInput>
> {
    protected modelName: string;
    protected prisma: PrismaClient;
    private createdIds: Set<string> = new Set();

    constructor(modelName: string, prisma: PrismaClient) {
        this.modelName = modelName;
        this.prisma = prisma;
    }

    // Abstract methods that subclasses must implement
    protected abstract getMinimalData(overrides?: Partial<TPrismaCreateInput>): TPrismaCreateInput;
    protected abstract getCompleteData(overrides?: Partial<TPrismaCreateInput>): TPrismaCreateInput;
    protected abstract getPrismaDelegate(): any;

    // Core factory methods

    /**
     * Create a minimal valid record
     */
    async createMinimal(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const data = this.getMinimalData(overrides);
        const result = await this.getPrismaDelegate().create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create a complete record with all optional fields populated
     */
    async createComplete(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const data = this.getCompleteData(overrides);
        const result = await this.getPrismaDelegate().create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create a record with complex relationships
     */
    async createWithRelations(config: RelationConfig): Promise<TPrismaModel> {
        return await this.prisma.$transaction(async (tx) => {
            // Get base data
            const baseData = this.getMinimalData(config.overrides);
            
            // Apply relationship configurations
            const dataWithRelations = await this.applyRelationships(baseData, config, tx);
            
            // Create the record
            const txDelegate = this.getTxPrismaDelegate(tx);
            const result = await txDelegate.create({ 
                data: dataWithRelations,
                include: this.getDefaultInclude(),
            });
            
            this.trackCreatedId(result.id);
            return result;
        });
    }

    // Bulk operations

    /**
     * Seed multiple records efficiently
     */
    async seedMultiple(count: number, template?: Partial<TPrismaCreateInput>): Promise<TPrismaModel[]> {
        const records: TPrismaModel[] = [];
        
        // Use createMany for efficiency when possible
        const dataArray = Array.from({ length: count }, (_, i) => {
            const baseData = this.getMinimalData(template);
            return this.makeUnique(baseData, i);
        });

        // If the model supports createMany, use it
        if (this.supportsCreateMany()) {
            await this.getPrismaDelegate().createMany({ data: dataArray });
            // Fetch the created records
            const ids = dataArray.map(d => (d as any).id);
            const created = await this.getPrismaDelegate().findMany({
                where: { id: { in: ids } },
            });
            created.forEach((r: any) => this.trackCreatedId(r.id));
            return created;
        } else {
            // Fall back to individual creates
            for (const data of dataArray) {
                const record = await this.getPrismaDelegate().create({ data });
                this.trackCreatedId(record.id);
                records.push(record);
            }
            return records;
        }
    }

    /**
     * Create a complex test scenario
     */
    async seedScenario(scenario: TestScenario): Promise<ScenarioResult> {
        return await this.prisma.$transaction(async (tx) => {
            const relatedIds: Record<string, string[]> = {};
            
            // Execute scenario setup
            const mainRecord = await this.createScenarioRecord(scenario, tx, relatedIds);
            
            return {
                id: mainRecord.id,
                scenario: scenario.name,
                relatedIds,
            };
        });
    }

    // Relationship management

    /**
     * Setup relationships for an existing record
     */
    async setupRelationships(parentId: string, config: RelationConfig): Promise<void> {
        await this.prisma.$transaction(async (tx) => {
            const txDelegate = this.getTxPrismaDelegate(tx);
            
            // Get current record
            const current = await txDelegate.findUnique({ 
                where: { id: parentId },
                include: this.getDefaultInclude(),
            });
            
            if (!current) {
                throw new Error(`${this.modelName} with id ${parentId} not found`);
            }
            
            // Apply relationships
            const updates = await this.buildRelationshipUpdates(config, tx);
            
            if (Object.keys(updates).length > 0) {
                await txDelegate.update({
                    where: { id: parentId },
                    data: updates,
                });
            }
        });
    }

    /**
     * Connect existing records via relationships
     */
    async connectExisting(id: string, relations: RelationConnections): Promise<void> {
        const updates: any = {};
        
        for (const [field, connection] of Object.entries(relations)) {
            if (Array.isArray(connection)) {
                updates[field] = {
                    connect: connection,
                };
            } else {
                updates[field] = {
                    connect: connection,
                };
            }
        }
        
        await this.getPrismaDelegate().update({
            where: { id },
            data: updates,
        });
    }

    // Verification utilities

    /**
     * Verify the state of a record matches expectations
     */
    async verifyState(id: string, expected: Partial<TPrismaModel>): Promise<void> {
        const actual = await this.getPrismaDelegate().findUnique({
            where: { id },
            include: this.getDefaultInclude(),
        });
        
        if (!actual) {
            throw new Error(`${this.modelName} with id ${id} not found`);
        }
        
        // Deep comparison of expected vs actual
        for (const [key, expectedValue] of Object.entries(expected)) {
            const actualValue = (actual as any)[key];
            
            if (JSON.stringify(actualValue) !== JSON.stringify(expectedValue)) {
                throw new Error(
                    `${this.modelName} state mismatch for field '${key}': ` +
                    `expected ${JSON.stringify(expectedValue)}, ` +
                    `got ${JSON.stringify(actualValue)}`,
                );
            }
        }
    }

    /**
     * Verify relationship counts
     */
    async verifyRelationships(id: string, expectedCounts: RelationCounts): Promise<void> {
        const record = await this.getPrismaDelegate().findUnique({
            where: { id },
            include: this.getRelationshipCountInclude(expectedCounts),
        });
        
        if (!record) {
            throw new Error(`${this.modelName} with id ${id} not found`);
        }
        
        for (const [relation, expectedCount] of Object.entries(expectedCounts)) {
            const actualCount = record._count?.[relation] ?? 0;
            
            if (actualCount !== expectedCount) {
                throw new Error(
                    `${this.modelName} relationship count mismatch for '${relation}': ` +
                    `expected ${expectedCount}, got ${actualCount}`,
                );
            }
        }
    }

    /**
     * Verify database constraints are properly enforced
     */
    async verifyConstraints(id: string): Promise<ConstraintValidation> {
        const violations: string[] = [];
        
        try {
            const record = await this.getPrismaDelegate().findUnique({
                where: { id },
                include: this.getDefaultInclude(),
            });
            
            if (!record) {
                violations.push("Record not found");
                return { valid: false, violations };
            }
            
            // Check model-specific constraints
            const modelViolations = await this.checkModelConstraints(record);
            violations.push(...modelViolations);
            
        } catch (error) {
            violations.push(`Database error: ${error instanceof Error ? error.message : "Unknown error"}`);
        }
        
        return {
            valid: violations.length === 0,
            violations,
        };
    }

    // Cleanup utilities

    /**
     * Clean up specific records by IDs
     */
    async cleanup(ids: string[]): Promise<void> {
        if (ids.length === 0) return;
        
        try {
            await this.getPrismaDelegate().deleteMany({
                where: { id: { in: ids } },
            });
            
            // Remove from tracking
            ids.forEach(id => this.createdIds.delete(id));
        } catch (error) {
            console.warn(`Failed to cleanup ${this.modelName} records:`, error);
        }
    }

    /**
     * Clean up all records created by this factory instance
     */
    async cleanupAll(): Promise<void> {
        const ids = Array.from(this.createdIds);
        await this.cleanup(ids);
    }

    /**
     * Clean up a record and all its related data
     */
    async cleanupRelated(id: string, depth = 1): Promise<void> {
        await this.prisma.$transaction(async (tx) => {
            await this.cascadeDelete(id, depth, tx);
        });
    }

    // Helper methods

    /**
     * Track created IDs for cleanup
     */
    protected trackCreatedId(id: string): void {
        this.createdIds.add(id);
    }

    /**
     * Get tracked IDs
     */
    getCreatedIds(): string[] {
        return Array.from(this.createdIds);
    }

    /**
     * Clear tracked IDs without cleanup
     */
    clearTracking(): void {
        this.createdIds.clear();
    }

    // Methods that can be overridden by subclasses

    /**
     * Get default include for queries
     */
    protected getDefaultInclude(): TPrismaInclude {
        return {} as TPrismaInclude;
    }

    /**
     * Apply relationships to create data
     */
    protected async applyRelationships(
        baseData: TPrismaCreateInput,
        config: RelationConfig,
        tx: PrismaClient,
    ): Promise<TPrismaCreateInput> {
        // Default implementation - subclasses should override
        return baseData;
    }

    /**
     * Build relationship updates
     */
    protected async buildRelationshipUpdates(
        config: RelationConfig,
        tx: PrismaClient,
    ): Promise<any> {
        // Default implementation - subclasses should override
        return {};
    }

    /**
     * Check model-specific constraints
     */
    protected async checkModelConstraints(record: TPrismaModel): Promise<string[]> {
        // Default implementation - subclasses should override
        return [];
    }

    /**
     * Make data unique for bulk operations
     */
    protected makeUnique(data: TPrismaCreateInput, index: number): TPrismaCreateInput {
        // Default implementation - just regenerate IDs
        return {
            ...data,
            id: generatePK(),
            publicId: generatePublicId(),
        } as TPrismaCreateInput;
    }

    /**
     * Check if model supports createMany
     */
    protected supportsCreateMany(): boolean {
        // Most models support createMany, override if not
        return true;
    }

    /**
     * Get Prisma delegate for transactions
     */
    protected getTxPrismaDelegate(tx: PrismaClient): any {
        return tx[this.modelName.toLowerCase()];
    }

    /**
     * Create a record for a scenario
     */
    protected async createScenarioRecord(
        scenario: TestScenario,
        tx: PrismaClient,
        relatedIds: Record<string, string[]>,
    ): Promise<any> {
        // Default implementation - subclasses should override
        const data = this.getMinimalData(scenario.config);
        const txDelegate = this.getTxPrismaDelegate(tx);
        return await txDelegate.create({ data });
    }

    /**
     * Get include for relationship counts
     */
    protected getRelationshipCountInclude(expectedCounts: RelationCounts): any {
        const include: any = {
            _count: {
                select: {},
            },
        };
        
        for (const relation of Object.keys(expectedCounts)) {
            include._count.select[relation] = true;
        }
        
        return include;
    }

    /**
     * Cascade delete implementation
     */
    protected async cascadeDelete(id: string, depth: number, tx: PrismaClient): Promise<void> {
        if (depth <= 0) {
            // Just delete this record
            const txDelegate = this.getTxPrismaDelegate(tx);
            await txDelegate.delete({ where: { id } });
            return;
        }
        
        // Get record with relationships
        const txDelegate = this.getTxPrismaDelegate(tx);
        const record = await txDelegate.findUnique({
            where: { id },
            include: this.getCascadeInclude(),
        });
        
        if (!record) return;
        
        // Delete related records first (subclasses should implement)
        await this.deleteRelatedRecords(record, depth - 1, tx);
        
        // Delete this record
        await txDelegate.delete({ where: { id } });
        this.createdIds.delete(id);
    }

    /**
     * Get include for cascade delete
     */
    protected getCascadeInclude(): any {
        // Subclasses should override to include relationships
        return {};
    }

    /**
     * Delete related records
     */
    protected async deleteRelatedRecords(
        record: TPrismaModel,
        remainingDepth: number,
        tx: PrismaClient,
    ): Promise<void> {
        // Subclasses should implement based on their relationships
    }
}
