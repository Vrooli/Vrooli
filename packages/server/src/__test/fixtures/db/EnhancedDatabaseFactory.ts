import { generatePK, generatePublicId } from "./idHelpers.js";
import { type PrismaClient } from "@prisma/client";
import type { 
    DatabaseFixtureFactory as IDatabaseFixtureFactory,
    RelationConfig,
    RelationConnections,
    RelationCounts,
    TestScenario,
    ScenarioResult,
    ConstraintValidation,
    DbResult,
    DbTestFixtures,
} from "./types.js";

/**
 * Enhanced database fixture factory that implements the ideal architecture
 * Provides comprehensive database testing capabilities with full type safety
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Complex relationship management
 * - Transaction-based test isolation
 * - Bulk operations and seeding
 * - State verification utilities
 * - Automatic cleanup tracking
 * - Scenario-based testing
 * 
 * @template TPrismaModel - The Prisma model type (e.g., User, Team)
 * @template TPrismaCreateInput - The Prisma create input type
 * @template TPrismaInclude - The Prisma include type for relationships
 * @template TPrismaUpdateInput - The Prisma update input type
 */
export abstract class EnhancedDatabaseFactory<
    TPrismaModel extends { id: string },
    TPrismaCreateInput,
    TPrismaInclude = any,
    TPrismaUpdateInput = Partial<TPrismaCreateInput>
> implements IDatabaseFixtureFactory<TPrismaCreateInput, TPrismaInclude> {
    
    protected modelName: string;
    protected prisma: PrismaClient;
    private createdIds: Set<string> = new Set();
    private static instances = new Map<string, EnhancedDatabaseFactory<any, any>>();

    constructor(modelName: string, prisma: PrismaClient) {
        this.modelName = modelName;
        this.prisma = prisma;
    }

    /**
     * Get singleton instance per model and prisma client
     */
    static getInstance<T extends EnhancedDatabaseFactory<any, any>>(
        this: new (modelName: string, prisma: PrismaClient) => T,
        modelName: string,
        prisma: PrismaClient,
    ): T {
        const key = `${modelName}_${prisma}`;
        if (!this.instances.has(key)) {
            this.instances.set(key, new this(modelName, prisma));
        }
        return this.instances.get(key) as T;
    }

    // Abstract methods that subclasses must implement

    /**
     * Get test fixtures for this model
     */
    protected abstract getFixtures(): DbTestFixtures<TPrismaCreateInput, TPrismaUpdateInput>;

    /**
     * Get the Prisma delegate for this model
     */
    protected abstract getPrismaDelegate(): any;

    /**
     * Get minimal valid data for this model
     */
    protected abstract generateMinimalData(overrides?: Partial<TPrismaCreateInput>): TPrismaCreateInput;

    /**
     * Get complete data with all fields populated
     */
    protected abstract generateCompleteData(overrides?: Partial<TPrismaCreateInput>): TPrismaCreateInput;

    // Core factory methods

    /**
     * Create a minimal valid record
     */
    async createMinimal(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const data = this.generateMinimalData(overrides);
        const result = await this.getPrismaDelegate().create({ 
            data,
            include: this.getDefaultInclude(),
        });
        this.trackCreatedId(result.id);
        return result as TPrismaModel;
    }

    /**
     * Create a complete record with all optional fields populated
     */
    async createComplete(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const data = this.generateCompleteData(overrides);
        const result = await this.getPrismaDelegate().create({ 
            data,
            include: this.getDefaultInclude(),
        });
        this.trackCreatedId(result.id);
        return result as TPrismaModel;
    }

    /**
     * Create a record with complex relationships
     */
    async createWithRelations(config: RelationConfig): Promise<TPrismaModel> {
        return await this.prisma.$transaction(async (tx) => {
            // Get base data
            const baseData = this.generateMinimalData(config.overrides);
            
            // Apply relationship configurations
            const dataWithRelations = await this.applyRelationships(baseData, config, tx);
            
            // Create the record
            const txDelegate = this.getTxPrismaDelegate(tx);
            const result = await txDelegate.create({ 
                data: dataWithRelations,
                include: this.getDefaultInclude(),
            });
            
            this.trackCreatedId(result.id);
            return result as TPrismaModel;
        });
    }

    // Bulk operations

    /**
     * Seed multiple records efficiently
     */
    async seedMultiple(count: number, template?: Partial<TPrismaCreateInput>): Promise<TPrismaModel[]> {
        const records: TPrismaModel[] = [];
        
        // Generate unique data for each record
        const dataArray = Array.from({ length: count }, (_, i) => {
            const baseData = this.generateMinimalData(template);
            return this.makeUnique(baseData, i);
        });

        // Use createMany for efficiency when possible
        if (this.supportsCreateMany() && dataArray.length > 1) {
            await this.getPrismaDelegate().createMany({ data: dataArray });
            
            // Fetch the created records
            const ids = dataArray.map(d => (d as any).id);
            const created = await this.getPrismaDelegate().findMany({
                where: { id: { in: ids } },
                include: this.getDefaultInclude(),
            });
            
            created.forEach((r: any) => this.trackCreatedId(r.id));
            return created as TPrismaModel[];
        } else {
            // Fall back to individual creates
            for (const data of dataArray) {
                const record = await this.getPrismaDelegate().create({ 
                    data,
                    include: this.getDefaultInclude(),
                });
                this.trackCreatedId(record.id);
                records.push(record as TPrismaModel);
            }
            return records;
        }
    }

    /**
     * Create a complex test scenario
     */
    async seedScenario(scenario: TestScenario | string): Promise<ScenarioResult> {
        const scenarioConfig = typeof scenario === "string" 
            ? this.getScenario(scenario)
            : scenario;

        return await this.prisma.$transaction(async (tx) => {
            const relatedIds: Record<string, string[]> = {};
            
            // Execute scenario setup
            const mainRecord = await this.createScenarioRecord(scenarioConfig, tx, relatedIds);
            
            return {
                id: mainRecord.id,
                scenario: scenarioConfig.name,
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
            const actualCount = (record as any)._count?.[relation] ?? 0;
            
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
    async cleanupRelated(id: string, options?: { depth?: number; include?: string[] }): Promise<void> {
        const depth = options?.depth ?? 1;
        const include = options?.include;
        
        await this.prisma.$transaction(async (tx) => {
            await this.cascadeDelete(id, depth, tx, include);
        });
    }

    // Test scenario management

    /**
     * Define test scenarios (override in subclasses)
     */
    protected scenarios: Record<string, TestScenario> = {};

    /**
     * Get a named scenario
     */
    protected getScenario(name: string): TestScenario {
        const scenario = this.scenarios[name];
        if (!scenario) {
            throw new Error(`Unknown scenario '${name}' for ${this.modelName}`);
        }
        return scenario;
    }

    /**
     * Get all available scenarios
     */
    getAvailableScenarios(): string[] {
        return Object.keys(this.scenarios);
    }

    // Factory methods for dynamic data generation

    /**
     * Create a factory function for this model
     */
    createFactory(defaults?: Partial<TPrismaCreateInput>): (overrides?: Partial<TPrismaCreateInput>) => Promise<TPrismaModel> {
        return async (overrides?: Partial<TPrismaCreateInput>) => {
            return this.createMinimal({ ...defaults, ...overrides });
        };
    }

    /**
     * Validate create input
     */
    async validateCreate(input: TPrismaCreateInput): Promise<{ valid: boolean; errors: string[] }> {
        try {
            // Attempt a dry run in a transaction that we'll rollback
            await this.prisma.$transaction(async (tx) => {
                const txDelegate = this.getTxPrismaDelegate(tx);
                await txDelegate.create({ data: input });
                throw new Error("ROLLBACK"); // Force rollback
            });
            return { valid: true, errors: [] };
        } catch (error) {
            if (error instanceof Error && error.message === "ROLLBACK") {
                return { valid: true, errors: [] };
            }
            return { 
                valid: false, 
                errors: [error instanceof Error ? error.message : "Unknown error"], 
            };
        }
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

    // Methods that subclasses should override

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
        tx: any,
    ): Promise<TPrismaCreateInput> {
        // Default implementation - subclasses should override
        return baseData;
    }

    /**
     * Build relationship updates
     */
    protected async buildRelationshipUpdates(
        config: RelationConfig,
        tx: any,
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
        const uniqueData = { ...data } as any;
        
        if ("id" in uniqueData) {
            uniqueData.id = generatePK();
        }
        
        if ("publicId" in uniqueData) {
            uniqueData.publicId = generatePublicId();
        }
        
        // Make handles unique if present
        if ("handle" in uniqueData && uniqueData.handle) {
            uniqueData.handle = `${uniqueData.handle}_${index}`;
        }
        
        // Make emails unique if present
        if ("email" in uniqueData && uniqueData.email) {
            const [localPart, domain] = uniqueData.email.split("@");
            uniqueData.email = `${localPart}+${index}@${domain}`;
        }
        
        return uniqueData as TPrismaCreateInput;
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
    protected getTxPrismaDelegate(tx: any): any {
        return tx[this.modelName.toLowerCase()];
    }

    /**
     * Create a record for a scenario
     */
    protected async createScenarioRecord(
        scenario: TestScenario,
        tx: any,
        relatedIds: Record<string, string[]>,
    ): Promise<TPrismaModel> {
        // Default implementation - subclasses should override for complex scenarios
        const data = this.generateMinimalData(scenario.config as Partial<TPrismaCreateInput>);
        const txDelegate = this.getTxPrismaDelegate(tx);
        const result = await txDelegate.create({ 
            data,
            include: this.getDefaultInclude(),
        });
        this.trackCreatedId(result.id);
        return result as TPrismaModel;
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
    protected async cascadeDelete(
        id: string, 
        depth: number, 
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        if (depth <= 0) {
            // Just delete this record
            const txDelegate = this.getTxPrismaDelegate(tx);
            await txDelegate.delete({ where: { id } });
            this.createdIds.delete(id);
            return;
        }
        
        // Get record with relationships
        const txDelegate = this.getTxPrismaDelegate(tx);
        const record = await txDelegate.findUnique({
            where: { id },
            include: this.getCascadeInclude(includeOnly),
        });
        
        if (!record) return;
        
        // Delete related records first (subclasses should implement)
        await this.deleteRelatedRecords(record, depth - 1, tx, includeOnly);
        
        // Delete this record
        await txDelegate.delete({ where: { id } });
        this.createdIds.delete(id);
    }

    /**
     * Get include for cascade delete
     */
    protected getCascadeInclude(includeOnly?: string[]): any {
        // Subclasses should override to include relationships
        return {};
    }

    /**
     * Delete related records
     */
    protected async deleteRelatedRecords(
        record: TPrismaModel,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Subclasses should implement based on their relationships
    }

    /**
     * Relationship handler registry
     */
    protected relationshipHandlers: Record<string, (parentId: string, config: any, tx: any) => Promise<void>> = {};

    /**
     * Register a relationship handler
     */
    protected registerRelationshipHandler(
        relationName: string,
        handler: (parentId: string, config: any, tx: any) => Promise<void>,
    ): void {
        this.relationshipHandlers[relationName] = handler;
    }

    /**
     * Execute relationship handlers
     */
    protected async executeRelationshipHandlers(
        parentId: string,
        config: RelationConfig,
        tx: any,
    ): Promise<void> {
        for (const [relation, relationConfig] of Object.entries(config)) {
            if (relation === "overrides") continue;
            
            const handler = this.relationshipHandlers[relation];
            if (handler) {
                await handler(parentId, relationConfig, tx);
            }
        }
    }
}
