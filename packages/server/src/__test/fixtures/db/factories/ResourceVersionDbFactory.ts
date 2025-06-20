import { generatePK, generatePublicId, ResourceType } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ResourceVersionRelationConfig extends RelationConfig {
    resource?: { id: string } | { create: Prisma.ResourceCreateInput };
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string; details?: string }>;
    relations?: Array<{
        toId: string;
        toType: "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "DataStructureVersion";
        index?: number;
    }>;
}

/**
 * Database fixture factory for ResourceVersion model
 * Handles versioned content and relationships to other versioned objects
 */
export class ResourceVersionDbFactory extends DatabaseFixtureFactory<
    Prisma.ResourceVersion,
    Prisma.ResourceVersionCreateInput,
    Prisma.ResourceVersionInclude,
    Prisma.ResourceVersionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ResourceVersion', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resourceVersion;
    }

    protected getMinimalData(overrides?: Partial<Prisma.ResourceVersionCreateInput>): Prisma.ResourceVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isComplete: false,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            complexity: 1,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Test Resource Version",
                    description: "A test resource version",
                }],
            },
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.ResourceVersionCreateInput>): Prisma.ResourceVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isComplete: true,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            versionNotes: "Complete version with all features",
            complexity: 5,
            link: "https://example.com/resource/v1",
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Resource Version",
                        description: "A comprehensive resource version",
                        instructions: "Follow these instructions to use this resource",
                        details: "Detailed information about this version",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Versión de Recurso Completa",
                        description: "Una versión de recurso integral",
                        instructions: "Sigue estas instrucciones para usar este recurso",
                        details: "Información detallada sobre esta versión",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.ResourceVersionInclude {
        return {
            translations: true,
            root: {
                select: {
                    id: true,
                    publicId: true,
                    resourceType: true,
                    isPrivate: true,
                    ownedByUser: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                            handle: true,
                        },
                    },
                    ownedByTeam: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            relations: {
                include: {
                    projectVersion: {
                        select: {
                            id: true,
                            publicId: true,
                            versionLabel: true,
                            translations: {
                                select: {
                                    language: true,
                                    name: true,
                                },
                            },
                        },
                    },
                    routineVersion: {
                        select: {
                            id: true,
                            publicId: true,
                            versionLabel: true,
                            translations: {
                                select: {
                                    language: true,
                                    name: true,
                                },
                            },
                        },
                    },
                },
            },
            _count: {
                select: {
                    relations: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.ResourceVersionCreateInput,
        config: ResourceVersionRelationConfig,
        tx: any
    ): Promise<Prisma.ResourceVersionCreateInput> {
        let data = { ...baseData };

        // Handle resource connection
        if (config.resource) {
            if ('id' in config.resource) {
                data.root = { connect: { id: config.resource.id } };
            } else if ('create' in config.resource) {
                data.root = { create: config.resource.create };
            }
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        // Handle relations to other versioned objects
        if (config.relations && Array.isArray(config.relations)) {
            data.relations = {
                create: config.relations.map((relation, idx) => ({
                    id: generatePK(),
                    index: relation.index ?? idx,
                    ...(relation.toType === "ProjectVersion" ? { projectVersionId: relation.toId } :
                        relation.toType === "RoutineVersion" ? { routineVersionId: relation.toId } :
                        relation.toType === "SmartContractVersion" ? { smartContractVersionId: relation.toId } :
                        relation.toType === "DataStructureVersion" ? { dataStructureVersionId: relation.toId } :
                        {}),
                })),
            };
        }

        return data;
    }

    /**
     * Create a documentation version
     */
    async createDocumentationVersion(resourceId: string): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            resource: { id: resourceId },
            translations: [{
                language: "en",
                name: "API Documentation v1.0",
                description: "Complete API reference guide",
                instructions: "Navigate through sections using the sidebar",
                details: "This documentation covers all endpoints, authentication methods, and examples",
            }],
            overrides: {
                link: "https://docs.example.com/api/v1",
                complexity: 3,
                isComplete: true,
            },
        });
    }

    /**
     * Create a tutorial version with relations
     */
    async createTutorialVersionWithRelations(
        resourceId: string,
        relatedVersions: Array<{ id: string; type: "ProjectVersion" | "RoutineVersion" }>
    ): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            resource: { id: resourceId },
            translations: [{
                language: "en",
                name: "Getting Started Tutorial",
                description: "Step-by-step guide for beginners",
                instructions: "Follow along with the examples in order",
                details: "Expected completion time: 30 minutes",
            }],
            relations: relatedVersions.map((rv, idx) => ({
                toId: rv.id,
                toType: rv.type,
                index: idx,
            })),
            overrides: {
                link: "https://tutorials.example.com/getting-started",
                complexity: 2,
                isComplete: true,
            },
        });
    }

    /**
     * Create a code repository version
     */
    async createCodeRepositoryVersion(resourceId: string): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            resource: { id: resourceId },
            translations: [{
                language: "en",
                name: "Example Code Repository",
                description: "Sample code implementations",
                instructions: "Clone the repository and follow README instructions",
                details: "Includes examples in TypeScript, Python, and Go",
            }],
            overrides: {
                link: "https://github.com/example/sample-code",
                versionLabel: "2.1.0",
                versionNotes: "Added TypeScript examples and improved documentation",
                complexity: 4,
                isComplete: true,
            },
        });
    }

    /**
     * Create a draft version (incomplete)
     */
    async createDraftVersion(resourceId: string): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            resource: { id: resourceId },
            translations: [{
                language: "en",
                name: "Work in Progress",
                description: "This resource is still being developed",
            }],
            overrides: {
                isComplete: false,
                isLatest: true,
                versionLabel: "0.1.0-draft",
                complexity: 1,
            },
        });
    }

    /**
     * Create a version with full metadata
     */
    async createWithFullMetadata(resourceId: string): Promise<Prisma.ResourceVersion> {
        return this.createWithRelations({
            resource: { id: resourceId },
            translations: [
                {
                    language: "en",
                    name: "Comprehensive Resource",
                    description: "A resource with complete metadata",
                    instructions: "1. Read overview\n2. Follow examples\n3. Complete exercises",
                    details: "This resource includes theory, practical examples, and self-assessment questions",
                },
                {
                    language: "es",
                    name: "Recurso Integral",
                    description: "Un recurso con metadatos completos",
                    instructions: "1. Leer resumen\n2. Seguir ejemplos\n3. Completar ejercicios",
                    details: "Este recurso incluye teoría, ejemplos prácticos y preguntas de autoevaluación",
                },
            ],
            overrides: {
                link: "https://resources.example.com/comprehensive",
                versionLabel: "3.0.0",
                versionNotes: "Major update with new content structure and interactive elements",
                complexity: 7,
                isComplete: true,
            },
        });
    }

    protected async checkModelConstraints(record: Prisma.ResourceVersion): Promise<string[]> {
        const violations: string[] = [];

        // Check version label format
        if (record.versionLabel && !/^\d+\.\d+\.\d+(-\w+)?$/.test(record.versionLabel)) {
            violations.push('Version label must follow semantic versioning format');
        }

        // Check complexity range
        if (record.complexity < 1 || record.complexity > 10) {
            violations.push('Complexity must be between 1 and 10');
        }

        // Check that if isLatest is true, no other version has isLatest for the same resource
        if (record.isLatest && record.rootId) {
            const otherLatest = await this.prisma.resourceVersion.count({
                where: {
                    rootId: record.rootId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            if (otherLatest > 0) {
                violations.push('Only one version can be marked as latest');
            }
        }

        // Check link format if provided
        if (record.link && !record.link.startsWith('http://') && !record.link.startsWith('https://')) {
            violations.push('Link must be a valid URL starting with http:// or https://');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing required fields
                isComplete: false,
                isLatest: true,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                isComplete: "yes", // Should be boolean
                isLatest: 1, // Should be boolean
                complexity: "high", // Should be number
                link: true, // Should be string
            },
            invalidVersionLabel: {
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "v1", // Invalid format
                complexity: 5,
            },
            invalidComplexity: {
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                complexity: 15, // Out of range (max 10)
            },
            invalidLink: {
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                complexity: 5,
                link: "not-a-url", // Invalid URL format
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.ResourceVersionCreateInput> {
        return {
            minimalComplexity: {
                ...this.getMinimalData(),
                complexity: 1,
            },
            maximalComplexity: {
                ...this.getMinimalData(),
                complexity: 10,
            },
            prereleaseVersion: {
                ...this.getMinimalData(),
                versionLabel: "2.0.0-beta.1",
                isComplete: false,
                isLatest: false,
            },
            longVersionNotes: {
                ...this.getMinimalData(),
                versionNotes: "This is a major update that includes numerous improvements and new features. ".repeat(20),
            },
            multiLanguageVersion: {
                ...this.getMinimalData(),
                translations: {
                    create: [
                        { id: generatePK(), language: "en", name: "Multi-language Resource", description: "Available in multiple languages" },
                        { id: generatePK(), language: "es", name: "Recurso Multiidioma", description: "Disponible en múltiples idiomas" },
                        { id: generatePK(), language: "fr", name: "Ressource Multilingue", description: "Disponible en plusieurs langues" },
                        { id: generatePK(), language: "de", name: "Mehrsprachige Ressource", description: "In mehreren Sprachen verfügbar" },
                        { id: generatePK(), language: "ja", name: "多言語リソース", description: "複数の言語で利用可能" },
                    ],
                },
            },
            manyRelations: {
                ...this.getMinimalData(),
                // Would need to add relations after creation in test
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            relations: {
                include: {
                    projectVersion: true,
                    routineVersion: true,
                    smartContractVersion: true,
                    dataStructureVersion: true,
                },
            },
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ResourceVersion,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete relations
        if (record.relations?.length) {
            await tx.resourceVersionRelation.deleteMany({
                where: { resourceVersionId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.resourceVersionTranslation.deleteMany({
                where: { resourceVersionId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createResourceVersionDbFactory = (prisma: PrismaClient) => new ResourceVersionDbFactory(prisma);