// AI_CHECK: TYPE_SAFETY=server-factory-bigint-migration | LAST: 2025-06-29 - Removed old interfaces, enforced BigInt IDs only
import { generatePK, generatePublicId } from "./idHelpers.js";
import { type PrismaClient } from "@prisma/client";
import type { RelationConfig, DbFactoryConfig } from "./types.js";
import { TestRecordTracker } from "../../helpers/testRecordTracker.js";

/**
 * Modern database fixture factory with BigInt ID support
 * 
 * Breaking Changes from Previous Version:
 * - IDs are now BigInt only (no string support)
 * - All Prisma types must use snake_case table names
 * - Removed legacy interfaces to force migration
 * 
 * Features:
 * - Type-safe BigInt ID handling
 * - Snake_case Prisma model support
 * - Relationship management
 * - Bulk operations
 * - Transaction support
 * 
 * @template TPrismaModel - The Prisma model type with BigInt ID
 * @template TPrismaCreateInput - The snake_case Prisma create input type
 * @template TPrismaInclude - The snake_case Prisma include type
 * @template TPrismaUpdateInput - The snake_case Prisma update input type
 */
export abstract class EnhancedDatabaseFactory<
    TPrismaModel extends { id: bigint },
    TPrismaCreateInput,
    TPrismaInclude = any,
    TPrismaUpdateInput = Partial<TPrismaCreateInput>
> {
    
    protected modelName: string;
    protected prisma: PrismaClient;
    private createdIds: Set<bigint> = new Set();
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
        if (!EnhancedDatabaseFactory.instances.has(key)) {
            EnhancedDatabaseFactory.instances.set(key, new this(modelName, prisma));
        }
        return EnhancedDatabaseFactory.instances.get(key) as T;
    }

    // Abstract methods that subclasses must implement

    /**
     * Get the Prisma delegate for this model (snake_case)
     * e.g., this.prisma.api_key, this.prisma.bookmark_list
     */
    protected abstract getPrismaDelegate(): any;

    /**
     * Generate minimal data for creating a record
     */
    protected abstract generateMinimalData(overrides?: Partial<TPrismaCreateInput>): TPrismaCreateInput;

    /**
     * Generate complete data for creating a record
     */
    protected abstract generateCompleteData(overrides?: Partial<TPrismaCreateInput>): TPrismaCreateInput;

    // Core factory methods

    /**
     * Create a minimal record
     */
    async createMinimal(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const data = this.generateMinimalData(overrides);
        if (!data) {
            throw new Error("Failed to generate minimal data");
        }
        
        const delegate = this.getPrismaDelegate();
        if (!delegate) {
            throw new Error("No Prisma delegate available");
        }
        
        const result = await delegate.create({ data });
        if (result && result.id) {
            this.createdIds.add(result.id);
            // Also track in global record tracker for coordinated cleanup
            TestRecordTracker.track(this.modelName.toLowerCase(), result.id);
        }
        return result;
    }

    /**
     * Create a complete record with all optional fields
     */
    async createComplete(overrides?: Partial<TPrismaCreateInput>): Promise<TPrismaModel> {
        const data = this.generateCompleteData(overrides);
        const delegate = this.getPrismaDelegate();
        const result = await delegate.create({ data });
        this.createdIds.add(result.id);
        // Also track in global record tracker for coordinated cleanup
        TestRecordTracker.track(this.modelName.toLowerCase(), result.id);
        return result;
    }

    /**
     * Create multiple records
     */
    async createMany(count: number, template?: Partial<TPrismaCreateInput>): Promise<TPrismaModel[]> {
        const records: TPrismaModel[] = [];
        for (let i = 0; i < count; i++) {
            const record = await this.createMinimal(template);
            records.push(record);
        }
        return records;
    }

    /**
     * Create a record with automatic relationship setup
     * Subclasses should override applyRelationships() to customize behavior
     */
    async createWithRelations(config: RelationConfig): Promise<TPrismaModel> {
        // Use transaction for atomicity
        return await this.prisma.$transaction(async (tx) => {
            // Generate base data
            const baseData = this.generateMinimalData(config.overrides);
            
            // Apply relationships if the subclass implements it
            let dataWithRelations = baseData;
            if (this.applyRelationships) {
                dataWithRelations = await this.applyRelationships(baseData, config, tx);
            }
            
            // Create the record with all relationships
            const delegate = this.getPrismaDelegate();
            const result = await delegate.create({
                data: dataWithRelations,
                ...(this.getDefaultInclude ? { include: this.getDefaultInclude() } : {}),
            });
            
            this.createdIds.add(result.id);
            // Also track in global record tracker for coordinated cleanup
            TestRecordTracker.track(this.modelName.toLowerCase(), result.id);
            return result;
        });
    }

    /**
     * Alias for createWithRelations to match naming conventions used elsewhere
     * Returns data with metadata structure to match EnhancedDbFactory interface
     */
    createWithRelationships(config: DbFactoryConfig<TPrismaCreateInput>): { data: TPrismaCreateInput; metadata: { hasAuth: boolean; teamCount: number; roleCount: number; relationCount: number } } {
        if (!config) {
            throw new Error("createWithRelationships requires a config parameter");
        }

        // Generate base data
        let data = this.generateMinimalData(config.overrides);
        if (!data) {
            throw new Error("Failed to create minimal data");
        }

        // Ensure the data is mutable
        if (Object.isFrozen(data)) {
            data = { ...data } as TPrismaCreateInput;
        }

        let hasAuth = false;
        let teamCount = 0;
        let roleCount = 0;
        let relationCount = 0;

        // For this synchronous version, we'll only apply the basic transformations
        // without database operations
        if (config.withAuth) {
            // Basic auth structure without database operations
            const authData = data as any;
            authData.auths = {
                create: [{
                    id: this.generateId(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                }],
            };
            authData.emails = {
                create: [{
                    id: this.generateId(),
                    emailAddress: `test_${Date.now()}@example.com`,
                    verifiedAt: new Date(),
                }],
            };
            hasAuth = true;
            relationCount++;
        }

        // Handle teams if provided
        if (config.withTeams && Array.isArray(config.withTeams)) {
            teamCount = config.withTeams.length;
            relationCount += teamCount;
        }

        // Handle roles if provided
        if (config.withRoles && Array.isArray(config.withRoles)) {
            roleCount = config.withRoles.length;
            relationCount += roleCount;
        }

        return {
            data,
            metadata: {
                hasAuth,
                teamCount,
                roleCount,
                relationCount,
            },
        };
    }

    /**
     * Find a record by ID
     */
    async findById(id: bigint, include?: TPrismaInclude): Promise<TPrismaModel | null> {
        const delegate = this.getPrismaDelegate();
        return await delegate.findUnique({
            where: { id },
            ...(include ? { include } : {}),
        });
    }

    /**
     * Update a record
     */
    async update(id: bigint, data: TPrismaUpdateInput): Promise<TPrismaModel> {
        const delegate = this.getPrismaDelegate();
        return await delegate.update({
            where: { id },
            data,
        });
    }

    /**
     * Delete a record
     */
    async delete(id: bigint): Promise<TPrismaModel> {
        const delegate = this.getPrismaDelegate();
        const result = await delegate.delete({
            where: { id },
        });
        this.createdIds.delete(id);
        // Also remove from global record tracker
        // Note: TestRecordTracker doesn't have a remove method, but cleanup() handles this
        return result;
    }

    /**
     * Clean up all created records
     */
    async cleanup(): Promise<void> {
        const delegate = this.getPrismaDelegate();
        if (this.createdIds.size > 0) {
            await delegate.deleteMany({
                where: {
                    id: {
                        in: Array.from(this.createdIds),
                    },
                },
            });
            this.createdIds.clear();
        }
    }

    /**
     * Enhanced cleanup that also clears global tracking
     * Used by DatabaseFactoryRegistry.cleanupAll()
     */
    async cleanupAll(): Promise<void> {
        await this.cleanup();
    }

    /**
     * Clear tracking data without deleting records
     * Used by DatabaseFactoryRegistry.clearAllTracking()
     */
    clearTracking(): void {
        this.createdIds.clear();
    }

    /**
     * Get all created IDs as string array for registry compatibility
     * Used by DatabaseFactoryRegistry.getAllCreatedIds()
     */
    getCreatedIds(): string[] {
        return Array.from(this.createdIds).map(id => id.toString());
    }

    /**
     * Verify database constraints for a specific record
     * Used by DatabaseFactoryRegistry.verifyIntegrity()
     */
    async verifyConstraints(id: string): Promise<{ valid: boolean; violations: string[] }> {
        const violations: string[] = [];
        
        try {
            const bigIntId = BigInt(id);
            const record = await this.findById(bigIntId);
            
            if (!record) {
                violations.push("Record not found");
            }
            
            // Subclasses can override this method to add specific constraint checks
            
        } catch (error) {
            violations.push(`Query error: ${error instanceof Error ? error.message : "Unknown error"}`);
        }
        
        return {
            valid: violations.length === 0,
            violations,
        };
    }

    /**
     * Setup relationships for an existing record
     * Used by DatabaseFactoryRegistry.createTestEnvironment()
     */
    async setupRelationships(id: string, config: any): Promise<void> {
        // Default implementation - subclasses should override for specific relationship setup
        const bigIntId = BigInt(id);
        
        if (this.applyRelationships) {
            // This would need additional implementation based on specific factory needs
            console.warn(`setupRelationships called on ${this.modelName} but not fully implemented`);
        }
    }

    /**
     * Generate a new BigInt ID
     */
    protected generateId(): bigint {
        return generatePK();
    }

    /**
     * Generate a new public ID string
     */
    protected generatePublicId(): string {
        return generatePublicId();
    }

    /**
     * Convert string ID to BigInt
     */
    protected stringToBigInt(id: string): bigint {
        return BigInt(id);
    }

    /**
     * Convert BigInt ID to string
     */
    protected bigIntToString(id: bigint): string {
        return id.toString();
    }

    // Optional methods that subclasses can override for createWithRelations support

    /**
     * Apply relationships to the base data
     * Override this in subclasses to handle model-specific relationships
     */
    protected applyRelationships?(
        baseData: TPrismaCreateInput,
        config: RelationConfig,
        tx: PrismaClient
    ): Promise<TPrismaCreateInput>;

    /**
     * Get default include configuration for queries
     * Override this in subclasses to define what relations to include by default
     */
    protected getDefaultInclude?(): TPrismaInclude;
}
