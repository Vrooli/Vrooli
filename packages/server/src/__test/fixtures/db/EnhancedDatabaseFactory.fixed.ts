/**
 * Fixed version of EnhancedDatabaseFactory
 * Resolves interface implementation issues and import problems
 */

import { type PrismaClient } from "@prisma/client";
import type { 
    DbTestFixtures,
    RelationConfig,
    RelationConnections,
    RelationCounts,
    TestScenario,
    ScenarioResult,
    ConstraintValidation,
    DbResult,
} from "./types.js";

// Temporary ID generation functions to work around import issues
// These will be replaced when import resolution is fixed
const generatePK = (): bigint => BigInt(Date.now() * 1000 + Math.floor(Math.random() * 1000));
const generatePublicId = (): string => Math.random().toString(36).substr(2, 9);

/**
 * Enhanced database fixture factory base class
 * Provides comprehensive database testing capabilities with full type safety
 * 
 * This version fixes the interface implementation issues and provides
 * a stable foundation for all model-specific factories
 */
export abstract class EnhancedDatabaseFactory<
    TPrismaModel extends { id: string | bigint },
    TPrismaCreateInput,
    TPrismaInclude = any,
    TPrismaUpdateInput = Partial<TPrismaCreateInput>
> {
    
    protected modelName: string;
    protected prisma: PrismaClient;
    private createdIds: Set<string | bigint> = new Set();

    constructor(modelName: string, prisma: PrismaClient) {
        this.modelName = modelName;
        this.prisma = prisma;
    }

    // Abstract methods that subclasses MUST implement

    /**
     * Get the Prisma delegate for this model
     * Example: return this.prisma.user;
     */
    protected abstract getPrismaDelegate(): any;

    /**
     * Get test fixtures for this model
     * This replaces the old generateMinimalData/generateCompleteData pattern
     */
    protected abstract getFixtures(): DbTestFixtures<TPrismaCreateInput, TPrismaUpdateInput>;

    // Core factory methods (implemented by base class)

    /**
     * Create a minimal valid record
     */
    async createMinimal(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const fixtures = this.getFixtures();
        const data = { ...fixtures.minimal, ...overrides };
        
        const result = await this.getPrismaDelegate().create({ 
            data,
            include: this.getDefaultInclude()
        });
        
        this.trackCreatedId(result.id);
        return result as TPrismaModel;
    }

    /**
     * Create a complete record with all optional fields populated
     */
    async createComplete(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const fixtures = this.getFixtures();
        const data = { ...fixtures.complete, ...overrides };
        
        const result = await this.getPrismaDelegate().create({ 
            data,
            include: this.getDefaultInclude()
        });
        
        this.trackCreatedId(result.id);
        return result as TPrismaModel;
    }

    /**
     * Create a record with complex relationships
     * Subclasses can override this for model-specific relationship logic
     */
    async createWithRelations(config: RelationConfig): Promise<TPrismaModel> {
        return await this.prisma.$transaction(async (tx) => {
            const fixtures = this.getFixtures();
            const baseData = { ...fixtures.minimal, ...config.overrides };
            
            // Apply any relationship configurations
            // This is a default implementation - subclasses should override for specific models
            
            const txDelegate = this.getTxPrismaDelegate(tx);
            const result = await txDelegate.create({ 
                data: baseData,
                include: this.getDefaultInclude()
            });
            
            this.trackCreatedId(result.id);
            return result as TPrismaModel;
        });
    }

    /**
     * Create multiple records efficiently
     */
    async seedMultiple(
        count: number, 
        template?: Partial<TPrismaCreateInput>
    ): Promise<TPrismaModel[]> {
        const records: TPrismaModel[] = [];
        const fixtures = this.getFixtures();
        
        for (let i = 0; i < count; i++) {
            const data = { 
                ...fixtures.minimal, 
                ...template,
                // Ensure unique IDs
                id: generatePK(),
            };
            
            const record = await this.getPrismaDelegate().create({ 
                data,
                include: this.getDefaultInclude()
            });
            
            this.trackCreatedId(record.id);
            records.push(record as TPrismaModel);
        }
        
        return records;
    }

    /**
     * Create a complex test scenario
     * Default implementation - subclasses can override for model-specific scenarios
     */
    async seedScenario(scenario: TestScenario | string): Promise<ScenarioResult> {
        // Default implementation creates a single minimal record
        const record = await this.createMinimal();
        
        return {
            id: record.id.toString(),
            scenario: typeof scenario === 'string' ? scenario : scenario.name,
            relatedIds: {}
        };
    }

    /**
     * Setup relationships for an existing record
     * Default implementation - subclasses should override for specific relationship logic
     */
    async setupRelationships(parentId: string, config: RelationConfig): Promise<void> {
        // Default implementation does nothing
        // Subclasses should implement specific relationship logic
    }

    /**
     * Connect existing records through relationships
     * Default implementation - subclasses should override for specific models
     */
    async connectExisting(id: string, relations: RelationConnections): Promise<void> {
        // Default implementation does nothing
        // Subclasses should implement specific connection logic
    }

    /**
     * Verify the state of a record
     */
    async verifyState(id: string, expected: Partial<DbResult>): Promise<void> {
        const record = await this.getPrismaDelegate().findUnique({
            where: { id: this.convertId(id) },
            include: this.getDefaultInclude()
        });
        
        if (!record) {
            throw new Error(`${this.modelName} with id ${id} not found`);
        }
        
        // Check each expected field
        for (const [key, expectedValue] of Object.entries(expected)) {
            const actualValue = (record as any)[key];
            if (actualValue !== expectedValue) {
                throw new Error(
                    `Field ${key} mismatch in ${this.modelName} ${id}: expected ${expectedValue}, got ${actualValue}`
                );
            }
        }
    }

    /**
     * Verify relationship counts for a record
     */
    async verifyRelationships(id: string, expectedCounts: RelationCounts): Promise<void> {
        const record = await this.getPrismaDelegate().findUnique({
            where: { id: this.convertId(id) },
            include: {
                _count: {
                    select: this.getCountSelections(expectedCounts)
                }
            }
        });
        
        if (!record) {
            throw new Error(`${this.modelName} with id ${id} not found`);
        }
        
        // Check each expected count
        for (const [relation, expectedCount] of Object.entries(expectedCounts)) {
            const actualCount = (record._count as any)[relation];
            if (actualCount !== expectedCount) {
                throw new Error(
                    `${relation} count mismatch in ${this.modelName} ${id}: expected ${expectedCount}, got ${actualCount}`
                );
            }
        }
    }

    /**
     * Verify database constraints for a record
     */
    async verifyConstraints(id: string): Promise<ConstraintValidation> {
        // Default implementation - basic existence check
        const record = await this.getPrismaDelegate().findUnique({
            where: { id: this.convertId(id) }
        });
        
        return {
            isValid: !!record,
            violations: record ? [] : [`${this.modelName} ${id} does not exist`],
            warnings: []
        };
    }

    /**
     * Clean up created records
     */
    async cleanup(ids: (string | bigint)[]): Promise<void> {
        if (ids.length === 0) return;
        
        const convertedIds = ids.map(id => this.convertId(id));
        
        await this.getPrismaDelegate().deleteMany({
            where: {
                id: {
                    in: convertedIds
                }
            }
        });
        
        // Remove from tracking
        ids.forEach(id => this.createdIds.delete(id));
    }

    /**
     * Clean up all records created by this factory instance
     */
    async cleanupAll(): Promise<void> {
        const ids = Array.from(this.createdIds);
        await this.cleanup(ids);
    }

    // Helper methods (can be overridden by subclasses)

    /**
     * Get default include configuration for queries
     * Subclasses can override to include default relationships
     */
    protected getDefaultInclude(): TPrismaInclude | undefined {
        return undefined;
    }

    /**
     * Get transaction version of Prisma delegate
     */
    protected getTxPrismaDelegate(tx: any) {
        return (tx as any)[this.modelName];
    }

    /**
     * Track created IDs for cleanup
     */
    protected trackCreatedId(id: string | bigint): void {
        this.createdIds.add(id);
    }

    /**
     * Convert string ID to appropriate type for the model
     * Override this if your model uses bigint IDs
     */
    protected convertId(id: string | bigint): string | bigint {
        // Default: try to convert to bigint if the original ID was bigint
        if (typeof id === 'bigint') return id;
        
        // Check if this looks like a bigint (all digits)
        if (typeof id === 'string' && /^\d+$/.test(id)) {
            try {
                return BigInt(id);
            } catch {
                return id;
            }
        }
        
        return id;
    }

    /**
     * Generate count selections based on expected counts
     */
    protected getCountSelections(expectedCounts: RelationCounts): Record<string, boolean> {
        const selections: Record<string, boolean> = {};
        for (const relation of Object.keys(expectedCounts)) {
            selections[relation] = true;
        }
        return selections;
    }

    /**
     * Get variant fixtures by name
     */
    getVariant(variantName: string): TPrismaCreateInput {
        const fixtures = this.getFixtures();
        if ('variants' in fixtures && fixtures.variants[variantName]) {
            return fixtures.variants[variantName];
        }
        throw new Error(`Variant '${variantName}' not found for ${this.modelName}`);
    }

    /**
     * Get invalid fixture by name
     */
    getInvalid(invalidName: string): Partial<TPrismaCreateInput> {
        const fixtures = this.getFixtures();
        if (fixtures.invalid[invalidName]) {
            return fixtures.invalid[invalidName];
        }
        throw new Error(`Invalid fixture '${invalidName}' not found for ${this.modelName}`);
    }

    /**
     * Get edge case fixture by name
     */
    getEdgeCase(edgeCaseName: string): TPrismaCreateInput {
        const fixtures = this.getFixtures();
        if ('edgeCase' in fixtures && fixtures.edgeCase[edgeCaseName]) {
            return fixtures.edgeCase[edgeCaseName];
        }
        throw new Error(`Edge case '${edgeCaseName}' not found for ${this.modelName}`);
    }

    /**
     * Validate fixture data
     */
    validateFixture(data: TPrismaCreateInput): { isValid: boolean; errors: string[] } {
        const errors: string[] = [];
        
        // Basic validation - check if required ID exists
        if (!data || typeof data !== 'object') {
            errors.push('Fixture data must be an object');
        } else if (!(data as any).id) {
            errors.push('Fixture data must include an id field');
        }
        
        return {
            isValid: errors.length === 0,
            errors
        };
    }
}