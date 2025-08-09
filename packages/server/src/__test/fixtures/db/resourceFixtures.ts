// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK, generatePublicId, ResourceType } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for Resource model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Cached IDs for consistent testing - lazy initialization pattern
let _resourceDbIds: Record<string, bigint> | null = null;
export function getResourceDbIds() {
    if (!_resourceDbIds) {
        _resourceDbIds = {
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
    }
    return _resourceDbIds;
}

/**
 * Minimal resource data for database creation
 */
export const minimalResourceDb: Prisma.resourceCreateInput = {
    id: getResourceDbIds().resource1,
    publicId: generatePublicId(),
    isPrivate: false,
    resourceType: ResourceType.Code,
};

/**
 * Resource with owner
 */
export const resourceWithOwnerDb: Prisma.resourceCreateInput = {
    id: getResourceDbIds().resource2,
    publicId: generatePublicId(),
    isPrivate: false,
    resourceType: ResourceType.Project,
    ownedByUser: {
        connect: { id: generatePK() }, // Will be replaced in factory
    },
};

/**
 * Complete resource with all features
 */
export const completeResourceDb: Prisma.resourceCreateInput = {
    id: getResourceDbIds().resource3,
    publicId: generatePublicId(),
    isPrivate: false,
    isInternal: false,
    resourceType: ResourceType.Routine,
    permissions: "{}",
    ownedByUser: {
        connect: { id: generatePK() }, // Will be replaced in factory
    },
    versions: {
        create: [{
            id: getResourceDbIds().version1,
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
                        id: getResourceDbIds().translation1,
                        language: "en",
                        name: "Test Resource",
                        description: "A complete test resource",
                        instructions: "Follow these steps to use the resource",
                        details: "Detailed information about the resource",
                    },
                    {
                        id: getResourceDbIds().translation2,
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
 * Enhanced test fixtures for Resource model following standard structure
 */
export const resourceDbFixtures: DbTestFixtures<Prisma.resourceCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        isPrivate: false,
        resourceType: ResourceType.Code,
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        isPrivate: false,
        isInternal: false,
        resourceType: ResourceType.Routine,
        permissions: JSON.stringify({
            canEdit: ["Owner", "Admin"],
            canView: ["Public"],
            canDelete: ["Owner"],
        }),
        ownedByUser: {
            connect: { id: generatePK() },
        },
        versions: {
            create: [{
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                versionNotes: "Initial complete version with all features",
                complexity: 8,
                translations: {
                    create: [
                        {
                            id: generatePK(),
                            language: "en",
                            name: "Complete Resource",
                            description: "A comprehensive resource with all features implemented",
                            instructions: "Follow these detailed steps to utilize this resource effectively",
                            details: "This resource includes advanced functionality and comprehensive documentation",
                        },
                        {
                            id: generatePK(),
                            language: "es",
                            name: "Recurso Completo",
                            description: "Un recurso integral con todas las características implementadas",
                            instructions: "Sigue estos pasos detallados para utilizar este recurso de manera efectiva",
                            details: "Este recurso incluye funcionalidad avanzada y documentación integral",
                        },
                    ],
                },
            }],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required resourceType
            isPrivate: false,
        },
        invalidTypes: {
            id: 123456789012345678n, // Invalid but properly typed bigint
            publicId: 123, // Should be string
            isPrivate: "yes", // Should be boolean
            isInternal: "no", // Should be boolean
            resourceType: "InvalidType", // Not a valid ResourceType
            permissions: { invalid: "object" }, // Should be string
        },
        invalidOwnership: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Code,
            ownedByUser: { connect: { id: generatePK() } },
            ownedByTeam: { connect: { id: generatePK() } }, // Both owners
        },
        invalidVersionData: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Routine,
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "", // Empty version label
                    complexity: -1, // Invalid complexity
                }],
            },
        },
    },
    edgeCases: {
        maxComplexityResource: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Code,
            ownedByUser: { connect: { id: generatePK() } },
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    complexity: 10, // Maximum complexity
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "Maximum Complexity Resource",
                            description: "A resource with the highest possible complexity rating",
                        }],
                    },
                }],
            },
        },
        multipleVersionsResource: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Api,
            ownedByUser: { connect: { id: generatePK() } },
            versions: {
                create: [
                    {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isComplete: true,
                        isLatest: false,
                        isPrivate: false,
                        versionLabel: "1.0.0",
                        complexity: 3,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                name: "API v1.0",
                                description: "Initial API version",
                            }],
                        },
                    },
                    {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isComplete: true,
                        isLatest: true,
                        isPrivate: false,
                        versionLabel: "2.0.0",
                        complexity: 6,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                name: "API v2.0",
                                description: "Enhanced API with new features",
                            }],
                        },
                    },
                ],
            },
        },
        privateInternalResource: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: true,
            isInternal: true,
            resourceType: ResourceType.Standard,
            ownedByTeam: { connect: { id: generatePK() } },
            permissions: JSON.stringify({
                canEdit: ["TeamOwner", "TeamAdmin"],
                canView: ["TeamMember"],
                canDelete: ["TeamOwner"],
            }),
        },
        multiLanguageResource: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Project,
            ownedByUser: { connect: { id: generatePK() } },
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    complexity: 5,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Global Project", description: "International collaboration project" },
                            { id: generatePK(), language: "es", name: "Proyecto Global", description: "Proyecto de colaboración internacional" },
                            { id: generatePK(), language: "fr", name: "Projet Global", description: "Projet de collaboration internationale" },
                            { id: generatePK(), language: "de", name: "Globales Projekt", description: "Internationales Kooperationsprojekt" },
                            { id: generatePK(), language: "ja", name: "グローバルプロジェクト", description: "国際協力プロジェクト" },
                        ],
                    },
                }],
            },
        },
    },
};

/**
 * Enhanced factory for creating resource database fixtures
 */
export class ResourceDbFactory extends EnhancedDbFactory<Prisma.resourceCreateInput> {

    /**
     * Override to exclude handle field which doesn't exist on resource model
     */
    protected generateFreshIdentifiers() {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            // No handle field for resource model
        };
    }

    /**
     * Get the test fixtures for Resource model
     */
    protected getFixtures(): DbTestFixtures<Prisma.resourceCreateInput> {
        return resourceDbFixtures;
    }

    /**
     * Get Resource-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getResourceDbIds().resource1, // Duplicate ID
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Code,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Code,
                    ownedByUser: { connect: { id: generatePK() } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    isPrivate: false,
                    resourceType: ResourceType.Code,
                },
            },
            validation: {
                requiredFieldMissing: resourceDbFixtures.invalid.missingRequired,
                invalidDataType: resourceDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Code,
                    versions: {
                        create: [{
                            id: generatePK(),
                            publicId: generatePublicId(),
                            isComplete: true,
                            isLatest: true,
                            isPrivate: false,
                            versionLabel: "1.0.0",
                            complexity: 15, // Out of range (max 10)
                        }],
                    },
                },
            },
            businessLogic: {
                bothUserAndTeamOwners: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Code,
                    ownedByUser: { connect: { id: generatePK() } },
                    ownedByTeam: { connect: { id: generatePK() } }, // Should have only one owner
                },
                privateResourceWithoutOwner: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: true, // Private but no owner
                    resourceType: ResourceType.Code,
                },
                incompleteLatestVersion: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Routine,
                    versions: {
                        create: [{
                            id: generatePK(),
                            publicId: generatePublicId(),
                            isComplete: false, // Latest version incomplete
                            isLatest: true,
                            isPrivate: false,
                            versionLabel: "1.0.0",
                            complexity: 5,
                        }],
                    },
                },
            },
        };
    }

    /**
     * Add owner to a resource fixture
     */
    protected addOwner(data: Prisma.resourceCreateInput, ownerId: bigint, ownerType: "user" | "team" = "user"): Prisma.resourceCreateInput {
        return {
            ...data,
            ...(ownerType === "user"
                ? { ownedByUser: { connect: { id: ownerId } } }
                : { ownedByTeam: { connect: { id: ownerId } } }
            ),
        };
    }

    /**
     * Add versions to a resource fixture
     */
    protected addVersions(data: Prisma.resourceCreateInput, versions: Array<{
        name: string;
        versionLabel: string;
        isLatest?: boolean;
        isComplete?: boolean;
        complexity?: number;
        description?: string;
    }>): Prisma.resourceCreateInput {
        return {
            ...data,
            versions: {
                create: versions.map(version => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: version.isComplete ?? false,
                    isLatest: version.isLatest ?? false,
                    isPrivate: false,
                    versionLabel: version.versionLabel,
                    complexity: version.complexity ?? 1,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: version.name,
                            description: version.description ?? "Resource version description",
                        }],
                    },
                })),
            },
        };
    }

    /**
     * Resource-specific validation
     */
    protected validateSpecific(data: Prisma.resourceCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Resource
        if (!data.resourceType) errors.push("Resource type is required");
        if (data.isPrivate === undefined) errors.push("isPrivate flag is required");

        // Check business logic
        if (data.ownedByUser && data.ownedByTeam) {
            errors.push("Resource cannot be owned by both user and team");
        }

        if (data.isPrivate && !data.ownedByUser && !data.ownedByTeam) {
            warnings.push("Private resource should have an owner");
        }

        if (data.isInternal && !data.ownedByTeam) {
            warnings.push("Internal resources are typically team-owned");
        }

        // Check versions
        if (data.versions?.create) {
            const versions = Array.isArray(data.versions.create) ? data.versions.create : [data.versions.create];
            const latestVersions = versions.filter(v => v.isLatest);

            if (latestVersions.length !== 1) {
                errors.push("Resource must have exactly one latest version");
            }

            const hasCompleteVersion = versions.some(v => v.isComplete);
            if (!hasCompleteVersion) {
                warnings.push("Resource should have at least one complete version");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        return factory.createMinimal(overrides);
    }

    static createWithOwner(
        ownerId: bigint,
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        const data = factory.createMinimal({ resourceType, ...overrides });
        return factory.addOwner(data, ownerId, "user");
    }

    static createWithTeamOwner(
        teamId: bigint,
        resourceType: ResourceType = ResourceType.Project,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        const data = factory.createMinimal({ resourceType, ...overrides });
        return factory.addOwner(data, teamId, "team");
    }

    static createComplete(
        ownerId: bigint,
        resourceType: ResourceType = ResourceType.Routine,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        const data = factory.createComplete({ resourceType, ...overrides });
        return factory.addOwner(data, ownerId, "user");
    }

    static createWithVersion(
        ownerId: bigint,
        versionData: {
            name: string;
            description?: string;
            versionLabel?: string;
            isComplete?: boolean;
        },
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        let data = factory.createMinimal({ resourceType, ...overrides });
        data = factory.addOwner(data, ownerId, "user");
        return factory.addVersions(data, [{
            name: versionData.name,
            versionLabel: versionData.versionLabel ?? "1.0.0",
            isLatest: true,
            isComplete: versionData.isComplete ?? false,
            complexity: 1,
            description: versionData.description,
        }]);
    }

    static createWithMultipleVersions(
        ownerId: bigint,
        versions: Array<{
            name: string;
            versionLabel: string;
            isLatest?: boolean;
            isComplete?: boolean;
            description?: string;
        }>,
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        let data = factory.createMinimal({ resourceType, ...overrides });
        data = factory.addOwner(data, ownerId, "user");
        return factory.addVersions(data, versions.map(version => ({
            name: version.name,
            versionLabel: version.versionLabel,
            isLatest: version.isLatest ?? false,
            isComplete: version.isComplete ?? false,
            complexity: 1,
            description: version.description,
        })));
    }

    /**
     * Create resource with specific permissions
     */
    static createWithPermissions(
        ownerId: bigint,
        permissions: Record<string, any>,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        const data = factory.createMinimal({
            resourceType: ResourceType.Code,
            permissions: JSON.stringify(permissions),
            ...overrides,
        });
        return factory.addOwner(data, ownerId, "user");
    }

    /**
     * Create private resource
     */
    static createPrivate(
        ownerId: bigint,
        resourceType: ResourceType = ResourceType.Code,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        const data = factory.createMinimal({
            resourceType,
            isPrivate: true,
            ...overrides,
        });
        return factory.addOwner(data, ownerId, "user");
    }

    /**
     * Create internal resource
     */
    static createInternal(
        ownerId: bigint,
        resourceType: ResourceType = ResourceType.Api,
        overrides?: Partial<Prisma.resourceCreateInput>,
    ): Prisma.resourceCreateInput {
        const factory = new ResourceDbFactory();
        const data = factory.createMinimal({
            resourceType,
            isInternal: true,
            ...overrides,
        });
        return factory.addOwner(data, ownerId, "user");
    }
}

/**
 * Factory for creating ResourceVersion fixtures
 */
export class ResourceVersionDbFactory {
    static createMinimal(
        resourceId: bigint,
        name: string,
        overrides?: Partial<Prisma.resource_versionCreateInput>,
    ): Prisma.resource_versionCreateInput {
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
        resourceId: bigint,
        name: string,
        overrides?: Partial<Prisma.resource_versionCreateInput>,
    ): Prisma.resource_versionCreateInput {
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
 * Enhanced helper to seed multiple test resources with comprehensive options
 */
export async function seedTestResources(
    prisma: any,
    ownerId: bigint,
    count = 3,
    options?: {
        resourceType?: ResourceType;
        withVersions?: boolean;
        isPrivate?: boolean;
        isInternal?: boolean;
        teamOwnerId?: bigint;
    },
): Promise<BulkSeedResult<any>> {
    const factory = new ResourceDbFactory();
    const resources = [];
    const resourceType = options?.resourceType ?? ResourceType.Code;
    let versionCount = 0;
    let privateCount = 0;
    let internalCount = 0;
    let teamOwnedCount = 0;

    for (let i = 0; i < count; i++) {
        let resourceData: Prisma.resourceCreateInput;
        const isPrivate = options?.isPrivate ?? false;
        const isInternal = options?.isInternal ?? false;

        if (options?.teamOwnerId) {
            resourceData = ResourceDbFactory.createWithTeamOwner(
                options.teamOwnerId,
                resourceType,
                {
                    isPrivate,
                    isInternal,
                },
            );
            teamOwnedCount++;
        } else {
            resourceData = options?.withVersions
                ? ResourceDbFactory.createWithVersion(
                    ownerId,
                    {
                        name: `Test Resource ${i + 1}`,
                        description: `Description for test resource ${i + 1}`,
                        isComplete: i % 2 === 0, // Alternate complete/incomplete
                    },
                    resourceType,
                    {
                        isPrivate,
                        isInternal,
                    },
                )
                : ResourceDbFactory.createWithOwner(ownerId, resourceType, {
                    isPrivate,
                    isInternal,
                });

            if (options?.withVersions) versionCount++;
        }

        if (isPrivate) privateCount++;
        if (isInternal) internalCount++;

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

    return {
        records: resources,
        summary: {
            total: resources.length,
            withVersions: versionCount,
            private: privateCount,
            internal: internalCount,
            teamOwned: teamOwnedCount,
            withAuth: 0,
            bots: 0,
            teams: teamOwnedCount,
        },
    };
}

/**
 * Helper to clean up test resources
 */
export async function cleanupTestResources(prisma: PrismaClient, resourceIds: bigint[]) {
    // Clean up in correct order for foreign keys
    await prisma.resource_version_translation.deleteMany({
        where: {
            resourceVersion: {
                root: {
                    id: { in: resourceIds },
                },
            },
        },
    });

    await prisma.resource_version.deleteMany({
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
