import { generatePK, generatePublicId, ResourceType } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Resource model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const resourceDbIds = {
    resource1: generatePK(),
    resource2: generatePK(),
    resource3: generatePK(),
    version1: generatePK(),
    version2: generatePK(),
    version3: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
};

/**
 * Minimal resource data for database creation
 */
export const minimalResourceDb: Prisma.ResourceCreateInput = {
    id: resourceDbIds.resource1,
    publicId: generatePublicId(),
    isPrivate: false,
    resourceType: ResourceType.Code,
};

/**
 * Resource with owner
 */
export const resourceWithOwnerDb: Prisma.ResourceCreateInput = {
    id: resourceDbIds.resource2,
    publicId: generatePublicId(),
    isPrivate: false,
    resourceType: ResourceType.Project,
    ownedByUser: {
        connect: { id: "user_owner_id" }, // Will be replaced in factory
    },
};

/**
 * Complete resource with all features
 */
export const completeResourceDb: Prisma.ResourceCreateInput = {
    id: resourceDbIds.resource3,
    publicId: generatePublicId(),
    isPrivate: false,
    isInternal: false,
    resourceType: ResourceType.Routine,
    permissions: "{}",
    ownedByUser: {
        connect: { id: "user_owner_id" }, // Will be replaced in factory
    },
    versions: {
        create: [{
            id: resourceDbIds.version1,
            publicId: generatePublicId(),
            isComplete: true,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            versionNotes: "Initial version",
            complexity: 5,
            translations: {
                create: [
                    {
                        id: resourceDbIds.translation1,
                        language: "en",
                        name: "Test Resource",
                        description: "A complete test resource",
                        instructions: "Follow these steps to use the resource",
                        details: "Detailed information about the resource",
                    },
                    {
                        id: resourceDbIds.translation2,
                        language: "es",
                        name: "Recurso de Prueba",
                        description: "Un recurso de prueba completo",
                        instructions: "Sigue estos pasos para usar el recurso",
                        details: "Información detallada sobre el recurso",
                    },
                ],
            },
        }],
    },
};

/**
 * Factory for creating resource database fixtures with overrides
 */
export class ResourceDbFactory {
    static createMinimal(overrides?: Partial<Prisma.ResourceCreateInput>): Prisma.ResourceCreateInput {
        return {
            ...minimalResourceDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    static createWithOwner(
        ownerId: string,
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...resourceWithOwnerDb,
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType,
            ownedByUser: { connect: { id: ownerId } },
            ...overrides,
        };
    }

    static createWithTeamOwner(
        teamId: string,
        resourceType: ResourceType = ResourceType.Project,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...minimalResourceDb,
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType,
            ownedByTeam: { connect: { id: teamId } },
            ...overrides,
        };
    }

    static createComplete(
        ownerId: string,
        resourceType: ResourceType = ResourceType.Routine,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        const versionId = generatePK();
        
        return {
            ...completeResourceDb,
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType,
            ownedByUser: { connect: { id: ownerId } },
            versions: {
                create: [{
                    id: versionId,
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionNotes: "Initial version",
                    complexity: 5,
                    translations: {
                        create: [
                            {
                                id: generatePK(),
                                language: "en",
                                name: "Test Resource",
                                description: "A complete test resource",
                                instructions: "Follow these steps to use the resource",
                                details: "Detailed information about the resource",
                            },
                        ],
                    },
                }],
            },
            ...overrides,
        };
    }

    static createWithVersion(
        ownerId: string,
        versionData: {
            name: string;
            description?: string;
            versionLabel?: string;
            isComplete?: boolean;
        },
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...this.createMinimal(overrides),
            resourceType,
            ownedByUser: { connect: { id: ownerId } },
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: versionData.isComplete ?? false,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: versionData.versionLabel ?? "1.0.0",
                    complexity: 1,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: versionData.name,
                            description: versionData.description ?? "Test resource description",
                        }],
                    },
                }],
            },
        };
    }

    static createWithMultipleVersions(
        ownerId: string,
        versions: Array<{
            name: string;
            versionLabel: string;
            isLatest?: boolean;
            isComplete?: boolean;
            description?: string;
        }>,
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...this.createMinimal(overrides),
            resourceType,
            ownedByUser: { connect: { id: ownerId } },
            versions: {
                create: versions.map(version => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: version.isComplete ?? false,
                    isLatest: version.isLatest ?? false,
                    isPrivate: false,
                    versionLabel: version.versionLabel,
                    complexity: 1,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: version.name,
                            description: version.description ?? "Test resource description",
                        }],
                    },
                })),
            },
        };
    }

    /**
     * Create resource with specific permissions
     */
    static createWithPermissions(
        ownerId: string,
        permissions: Record<string, any>,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...this.createWithOwner(ownerId, ResourceType.Code, overrides),
            permissions: JSON.stringify(permissions),
        };
    }

    /**
     * Create private resource
     */
    static createPrivate(
        ownerId: string,
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...this.createWithOwner(ownerId, resourceType, overrides),
            isPrivate: true,
        };
    }

    /**
     * Create internal resource
     */
    static createInternal(
        ownerId: string,
        resourceType: ResourceType = ResourceType.Api,
        overrides?: Partial<Prisma.ResourceCreateInput>
    ): Prisma.ResourceCreateInput {
        return {
            ...this.createWithOwner(ownerId, resourceType, overrides),
            isInternal: true,
        };
    }
}

/**
 * Factory for creating ResourceVersion fixtures
 */
export class ResourceVersionDbFactory {
    static createMinimal(
        resourceId: string,
        name: string,
        overrides?: Partial<Prisma.ResourceVersionCreateInput>
    ): Prisma.ResourceVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isComplete: false,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            complexity: 1,
            root: { connect: { id: resourceId } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name,
                    description: "Test resource version description",
                }],
            },
            ...overrides,
        };
    }

    static createComplete(
        resourceId: string,
        name: string,
        overrides?: Partial<Prisma.ResourceVersionCreateInput>
    ): Prisma.ResourceVersionCreateInput {
        return {
            ...this.createMinimal(resourceId, name, overrides),
            isComplete: true,
            versionNotes: "Complete version with all features",
            complexity: 5,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name,
                    description: "A complete resource version",
                    instructions: "Follow these instructions to use this resource",
                    details: "Detailed information about this version",
                }],
            },
        };
    }
}

/**
 * Helper to seed multiple test resources
 */
export async function seedTestResources(
    prisma: any,
    ownerId: string,
    count: number = 3,
    options?: {
        resourceType?: ResourceType;
        withVersions?: boolean;
        isPrivate?: boolean;
        isInternal?: boolean;
        teamOwnerId?: string;
    }
) {
    const resources = [];
    const resourceType = options?.resourceType ?? ResourceType.Code;

    for (let i = 0; i < count; i++) {
        let resourceData: Prisma.ResourceCreateInput;

        if (options?.teamOwnerId) {
            resourceData = ResourceDbFactory.createWithTeamOwner(
                options.teamOwnerId,
                resourceType,
                {
                    isPrivate: options?.isPrivate ?? false,
                    isInternal: options?.isInternal ?? false,
                }
            );
        } else {
            resourceData = options?.withVersions
                ? ResourceDbFactory.createWithVersion(
                    ownerId,
                    {
                        name: `Test Resource ${i + 1}`,
                        description: `Description for test resource ${i + 1}`,
                        isComplete: i % 2 === 0, // Alternate complete/incomplete
                    },
                    resourceType
                )
                : ResourceDbFactory.createWithOwner(ownerId, resourceType, {
                    isPrivate: options?.isPrivate ?? false,
                    isInternal: options?.isInternal ?? false,
                });
        }

        const resource = await prisma.resource.create({
            data: resourceData,
            include: {
                versions: {
                    include: {
                        translations: true,
                    },
                },
                ownedByUser: true,
                ownedByTeam: true,
            },
        });

        resources.push(resource);
    }

    return resources;
}

/**
 * Helper to clean up test resources
 */
export async function cleanupTestResources(prisma: any, resourceIds: string[]) {
    // Clean up in correct order for foreign keys
    await prisma.resourceVersionTranslation.deleteMany({
        where: {
            resourceVersion: {
                root: {
                    id: { in: resourceIds },
                },
            },
        },
    });

    await prisma.resourceVersion.deleteMany({
        where: {
            root: {
                id: { in: resourceIds },
            },
        },
    });

    await prisma.resource.deleteMany({
        where: { id: { in: resourceIds } },
    });
}