/* eslint-disable no-magic-numbers */
import { type Prisma, type PrismaClient, type resource_version } from "@prisma/client";
import { generatePublicId } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ResourceVersionRelationConfig extends RelationConfig {
    root?: { resourceId: bigint };
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string; details?: string }>;
    relations?: Array<{
        relationType: string;
        targetVersionId: bigint;
        isManualEntry?: boolean;
        sequence?: number;
    }>;
}

/**
 * Enhanced database fixture factory for ResourceVersion model
 * Handles versioned resource content with links, metadata, and translations
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Version management (latest/complete flags)
 * - Multi-language translations
 * - Resource version relationships
 * - Complexity ratings
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class ResourceVersionDbFactory extends EnhancedDatabaseFactory<
    resource_version,
    Prisma.resource_versionCreateInput,
    Prisma.resource_versionInclude,
    Prisma.resource_versionUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("ResourceVersion", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.resource_version;
    }

    /**
     * Get complete test fixtures for ResourceVersion model
     */
    protected getFixtures(): DbTestFixtures<Prisma.resource_versionCreateInput, Prisma.resource_versionUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: generatePublicId(),
                isComplete: false,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                complexity: 1,
            },
            complete: {
                id: this.generateId(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "2.0.0",
                versionNotes: "Major update with breaking changes",
                link: "https://example.com/resource/v2",
                complexity: 5,
                resourceIndex: 1,
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            name: "Complete Resource Version",
                            description: "A comprehensive resource with all features",
                            instructions: "Follow these steps to use the resource",
                            details: "Detailed information about the resource",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            name: "Versión Completa del Recurso",
                            description: "Un recurso integral con todas las características",
                            instructions: "Sigue estos pasos para usar el recurso",
                            details: "Información detallada sobre el recurso",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, versionLabel, complexity
                    isComplete: false,
                    isLatest: true,
                    isPrivate: false,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    isComplete: "yes", // Should be boolean
                    isLatest: "true", // Should be boolean
                    isPrivate: 1, // Should be boolean
                    versionLabel: 123, // Should be string
                    complexity: "five", // Should be number
                    link: 123, // Should be string
                },
                invalidComplexity: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    complexity: 11, // Should be 1-10
                },
            },
            edgeCases: {
                minimalComplexity: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    complexity: 1, // Minimum complexity
                },
                maximalComplexity: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    complexity: 10, // Maximum complexity
                },
                preReleaseVersion: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    isComplete: false,
                    isLatest: false,
                    isPrivate: true,
                    versionLabel: "3.0.0-beta.1",
                    versionNotes: "Beta release - not for production use",
                    complexity: 7,
                },
                deprecatedVersion: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: false,
                    isPrivate: false,
                    versionLabel: "0.9.0",
                    versionNotes: "Deprecated - please upgrade to v2.0.0",
                    complexity: 3,
                },
                multiLanguageVersion: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    complexity: 5,
                    translations: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: this.generateId(),
                            language: ["en", "es", "fr", "de", "ja"][i],
                            name: `Resource Version ${i}`,
                            description: `Description in language ${i}`,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    versionNotes: "Updated version notes",
                },
                complete: {
                    isComplete: true,
                    versionNotes: "Major update with new features",
                    link: "https://example.com/resource/v2-updated",
                    complexity: 8,
                    // Note: Translation updates would require existing translation IDs
                    // This is left empty as update operations need real existing IDs
                    // translations: {
                    //     update: [{
                    //         where: { id: existingTranslationId },
                    //         data: {
                    //             description: "Updated description",
                    //             instructions: "Updated instructions",
                    //         },
                    //     }],
                    // },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.resource_versionCreateInput>): Prisma.resource_versionCreateInput {
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            isComplete: false,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            complexity: 1,
            root: {
                connect: { id: this.generateId() },
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.resource_versionCreateInput>): Prisma.resource_versionCreateInput {
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            isComplete: true,
            isLatest: true,
            isPrivate: false,
            versionLabel: "2.0.0",
            versionNotes: "Major update with breaking changes",
            link: "https://example.com/resource/v2",
            complexity: 5,
            resourceIndex: 1,
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        name: "Complete Resource Version",
                        description: "A comprehensive resource with all features",
                        instructions: "Follow these steps to use the resource",
                        details: "Detailed information about the resource",
                    },
                    {
                        id: this.generateId(),
                        language: "es",
                        name: "Versión Completa del Recurso",
                        description: "Un recurso integral con todas las características",
                        instructions: "Sigue estos pasos para usar el recurso",
                        details: "Información detallada sobre el recurso",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            latestVersion: {
                name: "latestVersion",
                description: "Latest version of a resource",
                config: {
                    overrides: {
                        isLatest: true,
                        isComplete: true,
                        versionLabel: "2.0.0",
                    },
                    translations: [{
                        language: "en",
                        name: "Latest Version",
                        description: "The most recent version of this resource",
                    }],
                },
            },
            betaVersion: {
                name: "betaVersion",
                description: "Beta/pre-release version",
                config: {
                    overrides: {
                        isLatest: false,
                        isComplete: false,
                        isPrivate: true,
                        versionLabel: "3.0.0-beta.1",
                        versionNotes: "Beta release - not for production use",
                    },
                    translations: [{
                        language: "en",
                        name: "Beta Version",
                        description: "Pre-release version for testing",
                        instructions: "This is a beta version. Expect bugs and breaking changes.",
                    }],
                },
            },
            documentationVersion: {
                name: "documentationVersion",
                description: "Documentation resource version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        link: "https://docs.example.com/api/v2",
                        complexity: 3,
                    },
                    translations: [{
                        language: "en",
                        name: "API Documentation v2",
                        description: "Complete API reference documentation",
                        instructions: "Browse the API endpoints and examples",
                        details: "This documentation covers REST API v2 endpoints",
                    }],
                },
            },
            tutorialVersion: {
                name: "tutorialVersion",
                description: "Tutorial resource version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        link: "https://tutorials.example.com/getting-started",
                        complexity: 2,
                    },
                    translations: [{
                        language: "en",
                        name: "Getting Started Tutorial",
                        description: "Step-by-step tutorial for beginners",
                        instructions: "Follow along with the video and code examples",
                        details: "Duration: 30 minutes. Prerequisites: Basic programming knowledge",
                    }],
                },
            },
            complexVersion: {
                name: "complexVersion",
                description: "High complexity resource version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        complexity: 10,
                        versionLabel: "1.0.0",
                    },
                    translations: [{
                        language: "en",
                        name: "Advanced Machine Learning Model",
                        description: "Complex ML model implementation",
                        instructions: "Requires advanced knowledge of ML and mathematics",
                        details: "This implementation uses cutting-edge techniques and requires GPU acceleration",
                    }],
                    relations: [
                        {
                            relationType: "DependsOn",
                            targetVersionId: this.generateId(),
                            isManualEntry: false,
                            sequence: 1,
                        },
                        {
                            relationType: "DependsOn",
                            targetVersionId: this.generateId(),
                            isManualEntry: false,
                            sequence: 2,
                        },
                    ],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.resource_versionInclude {
        return {
            root: {
                select: {
                    id: true,
                    publicId: true,
                    resourceType: true,
                    isPrivate: true,
                },
            },
            translations: true,
            relations: {
                include: {
                    to: {
                        select: {
                            id: true,
                            publicId: true,
                            versionLabel: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    translations: true,
                    relations: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.resource_versionCreateInput,
        config: ResourceVersionRelationConfig,
        tx: any,
    ): Promise<Prisma.resource_versionCreateInput> {
        const data = { ...baseData };

        // Handle root resource connection
        if (config.root?.resourceId) {
            data.root = { connect: { id: config.root.resourceId } };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: this.generateId(),
                    ...trans,
                })),
            };
        }

        // Handle relations to other resource versions
        if (config.relations && Array.isArray(config.relations)) {
            data.relations = {
                create: config.relations.map(relation => ({
                    id: this.generateId(),
                    fromVersionId: data.id as bigint,
                    toVersionId: relation.targetVersionId,
                    relationType: relation.relationType,
                    labels: [],
                })),
            };
        }

        return data;
    }

    /**
     * Create a documentation version
     */
    async createDocumentationVersion(resourceId: bigint): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            root: { resourceId },
            overrides: {
                isComplete: true,
                isLatest: true,
                link: "https://docs.example.com/api/v2",
                complexity: 3,
                simplicity: 7,
            },
            translations: [{
                language: "en",
                name: "API Documentation v2",
                description: "Complete API reference documentation",
                instructions: "Browse the API endpoints and examples",
                details: "This documentation covers REST API v2 endpoints",
            }],
        });
    }

    /**
     * Create a tutorial version
     */
    async createTutorialVersion(resourceId: bigint): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            root: { resourceId },
            overrides: {
                isComplete: true,
                isLatest: true,
                link: "https://tutorials.example.com/getting-started",
                complexity: 2,
                simplicity: 8,
            },
            translations: [{
                language: "en",
                name: "Getting Started Tutorial",
                description: "Step-by-step tutorial for beginners",
                instructions: "Follow along with the video and code examples",
                details: "Duration: 30 minutes. Prerequisites: Basic programming knowledge",
            }],
        });
    }

    /**
     * Create a beta/pre-release version
     */
    async createBetaVersion(resourceId: bigint): Promise<Prisma.ResourceVersion> {
        return this.seedScenario("betaVersion");
    }

    /**
     * Create version with dependencies
     */
    async createVersionWithDependencies(
        resourceId: bigint,
        dependencyIds: (string | bigint)[],
    ): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            root: { resourceId },
            relations: dependencyIds.map((depId, index) => ({
                relationType: "DependsOn",
                targetVersionId: typeof depId === 'string' ? BigInt(depId) : depId,
                isManualEntry: false,
                sequence: index + 1,
            })),
            overrides: {
                isComplete: true,
                isLatest: true,
                complexity: 7,
            },
            translations: [{
                language: "en",
                name: "Version with Dependencies",
                description: "This version depends on other resources",
            }],
        });
    }

    protected async checkModelConstraints(record: Prisma.ResourceVersion): Promise<string[]> {
        const violations: string[] = [];

        // Check complexity range
        if (record.complexity < 1 || record.complexity > 10) {
            violations.push("Complexity must be between 1 and 10");
        }


        // Check that version belongs to a resource
        if (!record.rootId) {
            violations.push("ResourceVersion must belong to a Resource");
        }

        // Check version label format
        if (record.versionLabel && !/^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$/.test(record.versionLabel)) {
            violations.push("Version label should follow semantic versioning format");
        }

        // Check that only one version is marked as latest per resource
        if (record.isLatest && record.rootId) {
            const otherLatest = await this.prisma.resource_version.count({
                where: {
                    rootId: record.rootId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            if (otherLatest > 0) {
                violations.push("Only one version can be marked as latest per resource");
            }
        }

        // Check link format if provided
        if (record.link && !record.link.startsWith("http")) {
            violations.push("Link must be a valid URL");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            relations: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ResourceVersion,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) =>
            !includeOnly || includeOnly.includes(relation);

        // Delete translations
        if (shouldDelete("translations") && record.translations?.length) {
            await tx.resource_version_translation.deleteMany({
                where: { resourceVersionId: record.id },
            });
        }

        // Delete relations
        if (shouldDelete("relations") && record.relations?.length) {
            await tx.resource_version_relation.deleteMany({
                where: { OR: [{ fromId: record.id }, { toId: record.id }] },
            });
        }
    }

    /**
     * Create version history for testing
     */
    async createVersionHistory(resourceId: bigint, count = 3): Promise<Prisma.ResourceVersion[]> {
        const versions: Prisma.ResourceVersion[] = [];

        for (let i = 0; i < count; i++) {
            const isLatest = i === count - 1;
            const version = await this.createWithRelations({
                root: { resourceId },
                overrides: {
                    versionLabel: `${i + 1}.0.0`,
                    isLatest,
                    isComplete: true,
                    complexity: Math.min(10, i + 3),
                    versionNotes: isLatest ? "Latest version" : `Version ${i + 1} release notes`,
                },
                translations: [{
                    language: "en",
                    name: `Version ${i + 1}.0.0`,
                    description: isLatest ? "Latest version with all features" : `Version ${i + 1} of the resource`,
                }],
            });
            versions.push(version);
        }

        return versions;
    }
}

// Export factory creator function
export const createResourceVersionDbFactory = (prisma: PrismaClient) =>
    new ResourceVersionDbFactory(prisma);

// Export the class for type usage
export { ResourceVersionDbFactory as ResourceVersionDbFactoryClass };
