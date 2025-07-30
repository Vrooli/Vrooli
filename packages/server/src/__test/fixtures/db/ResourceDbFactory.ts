// AI_CHECK: TYPE_SAFETY=server-factory-bigint-migration | LAST: 2025-06-29 - Migrated to BigInt IDs, snake_case tables, correct field names
import { type Prisma, type PrismaClient, type resource } from "@prisma/client";
import { ResourceType } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";

/**
 * Enhanced database fixture factory for Resource model
 * Uses snake_case table name: 'resource'
 * 
 * Breaking Changes:
 * - IDs are now BigInt only
 * - Uses snake_case Prisma types (resourceCreateInput, resourceInclude)
 * - Uses correct field names from schema: publicId, resourceType, isPrivate, hasCompleteVersion, isDeleted
 * - Fixed ResourceType enum values to match current implementation
 * - All table references use snake_case
 * 
 * Features:
 * - Support for various resource types (Code, Note, Api, Project, Routine, Standard)
 * - Version management with resource_version
 * - Owner relationships (user or team)
 * - Tag associations
 * - Type-safe BigInt ID handling
 */
export class ResourceDbFactory extends EnhancedDatabaseFactory<
    resource,
    Prisma.resourceCreateInput,
    Prisma.resourceInclude,
    Prisma.resourceUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("resource", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resource;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        return {
            id: this.generateId(),
            publicId: this.generatePublicId(),
            resourceType: ResourceType.Code,
            isPrivate: false,
            hasCompleteVersion: false,
            isDeleted: false,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        return {
            id: this.generateId(),
            publicId: this.generatePublicId(),
            resourceType: ResourceType.Note,
            isPrivate: false,
            hasCompleteVersion: true,
            isDeleted: false,
            completedAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Create resource with versions
     */
    async createWithVersions(
        versionCount = 1,
        resourceOverrides?: Partial<Prisma.resourceCreateInput>,
    ): Promise<{
        resource: any;
        versions: Array<any>;
    }> {
        const resource = await this.createMinimal(resourceOverrides);
        const versions: Array<any> = [];

        for (let i = 0; i < versionCount; i++) {
            const version = await this.prisma.resource_version.create({
                data: {
                    id: this.generateId(),
                    publicId: this.generatePublicId(),
                    rootId: resource.id,
                    versionLabel: `v1.${i}`,
                    versionIndex: i,
                    isComplete: true,
                },
            });
            versions.push(version);
        }

        return { resource, versions };
    }

    /**
     * Create different resource types for testing
     */
    async createByType(type: ResourceType, overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createMinimal({
            resourceType: type,
            ...overrides,
        });
    }

    /**
     * Create API resource
     */
    async createApiResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createByType(ResourceType.Api, overrides);
    }

    /**
     * Create Code resource  
     */
    async createCodeResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createByType(ResourceType.Code, overrides);
    }

    /**
     * Create Note resource (formerly Documentation)
     */
    async createNoteResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createByType(ResourceType.Note, overrides);
    }

    /**
     * Create Project resource
     */
    async createProjectResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createByType(ResourceType.Project, overrides);
    }

    /**
     * Create Routine resource
     */
    async createRoutineResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createByType(ResourceType.Routine, overrides);
    }

    /**
     * Create Standard resource (formerly Tutorial)
     */
    async createStandardResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createByType(ResourceType.Standard, overrides);
    }

    /**
     * Create private resource
     */
    async createPrivateResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createMinimal({
            isPrivate: true,
            ...overrides,
        });
    }

    /**
     * Create completed resource
     */
    async createCompletedResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createMinimal({
            hasCompleteVersion: true,
            completedAt: new Date(),
            ...overrides,
        });
    }

    /**
     * Create deleted resource
     */
    async createDeletedResource(overrides?: Partial<Prisma.resourceCreateInput>) {
        return await this.createMinimal({
            isDeleted: true,
            ...overrides,
        });
    }
}

// Export factory creator function
export function createResourceDbFactory(prisma: PrismaClient) {
    return new ResourceDbFactory(prisma);
}
