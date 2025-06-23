import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
import { routineConfigFixtures } from "@vrooli/shared/test-fixtures";

interface RoutineVersionRelationConfig extends RelationConfig {
    routine?: { id: string } | { create: Prisma.RoutineCreateInput };
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    resourceLists?: Array<{
        listFor: string;
        resources?: Array<{
            name: string;
            link: string;
            usedFor?: string;
            translations?: Array<{ language: string; name: string; description?: string }>;
        }>;
    }>;
    runs?: number;
}

/**
 * Database fixture factory for RoutineVersion model
 * Handles version management, configurations, and complex graph structures
 */
export class RoutineVersionDbFactory extends DatabaseFixtureFactory<
    Prisma.RoutineVersion,
    Prisma.RoutineVersionCreateInput,
    Prisma.RoutineVersionInclude,
    Prisma.RoutineVersionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("RoutineVersion", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.routineVersion;
    }

    protected getMinimalData(overrides?: Partial<Prisma.RoutineVersionCreateInput>): Prisma.RoutineVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isComplete: false,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            complexity: 1,
            timesStarted: 0,
            timesCompleted: 0,
            config: routineConfigFixtures.minimal,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Test Routine Version",
                    description: "A test routine version",
                }],
            },
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.RoutineVersionCreateInput>): Prisma.RoutineVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isComplete: true,
            isLatest: true,
            isPrivate: false,
            versionLabel: "1.0.0",
            versionNotes: "Complete version with all features",
            complexity: 5,
            timesStarted: 10,
            timesCompleted: 8,
            config: routineConfigFixtures.complete,
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Routine Version",
                        description: "A comprehensive routine version with all features",
                        instructions: "Follow these detailed steps to execute the routine",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Versión de Rutina Completa",
                        description: "Una versión de rutina integral con todas las funcionalidades",
                        instructions: "Sigue estos pasos detallados para ejecutar la rutina",
                    },
                ],
            },
            resourceLists: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    listFor: "Learn",
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "Learning Resources",
                            description: "Resources to learn about this routine",
                        }],
                    },
                    resources: {
                        create: [{
                            id: generatePK(),
                            index: 0,
                            usedFor: "Tutorial",
                            resourceVersion: {
                                create: {
                                    id: generatePK(),
                                    publicId: generatePublicId(),
                                    isComplete: true,
                                    isLatest: true,
                                    isPrivate: false,
                                    versionLabel: "1.0.0",
                                    link: "https://example.com/tutorial",
                                    translations: {
                                        create: [{
                                            id: generatePK(),
                                            language: "en",
                                            name: "Tutorial Resource",
                                            description: "Learn how to use this routine",
                                        }],
                                    },
                                    root: {
                                        create: {
                                            id: generatePK(),
                                            publicId: generatePublicId(),
                                            isPrivate: false,
                                            resourceType: "Tutorial",
                                        },
                                    },
                                },
                            },
                        }],
                    },
                }],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.RoutineVersionInclude {
        return {
            translations: true,
            root: {
                select: {
                    id: true,
                    publicId: true,
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
            resourceLists: {
                include: {
                    translations: true,
                    resources: {
                        include: {
                            resourceVersion: {
                                include: {
                                    translations: true,
                                },
                            },
                        },
                    },
                },
            },
            _count: {
                select: {
                    runs: true,
                    nodeLinks: true,
                    nodes: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.RoutineVersionCreateInput,
        config: RoutineVersionRelationConfig,
        tx: any,
    ): Promise<Prisma.RoutineVersionCreateInput> {
        const data = { ...baseData };

        // Handle routine connection
        if (config.routine) {
            if ("id" in config.routine) {
                data.root = { connect: { id: config.routine.id } };
            } else if ("create" in config.routine) {
                data.root = { create: config.routine.create };
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

        // Handle resource lists
        if (config.resourceLists && Array.isArray(config.resourceLists)) {
            data.resourceLists = {
                create: config.resourceLists.map(list => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    listFor: list.listFor,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: `${list.listFor} Resources`,
                            description: `Resources for ${list.listFor}`,
                        }],
                    },
                    resources: list.resources ? {
                        create: list.resources.map((resource, index) => ({
                            id: generatePK(),
                            index,
                            usedFor: resource.usedFor,
                            resourceVersion: {
                                create: {
                                    id: generatePK(),
                                    publicId: generatePublicId(),
                                    isComplete: true,
                                    isLatest: true,
                                    isPrivate: false,
                                    versionLabel: "1.0.0",
                                    link: resource.link,
                                    translations: {
                                        create: resource.translations?.map(trans => ({
                                            id: generatePK(),
                                            ...trans,
                                        })) || [{
                                            id: generatePK(),
                                            language: "en",
                                            name: resource.name,
                                            description: `Resource: ${resource.name}`,
                                        }],
                                    },
                                    root: {
                                        create: {
                                            id: generatePK(),
                                            publicId: generatePublicId(),
                                            isPrivate: false,
                                            resourceType: resource.usedFor || "Link",
                                        },
                                    },
                                },
                            },
                        })),
                    } : undefined,
                })),
            };
        }

        return data;
    }

    /**
     * Create a simple action routine version
     */
    async createSimpleAction(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            routine: { id: routineId },
            translations: [{
                language: "en",
                name: "Simple Action Version",
                description: "Execute a simple action",
                instructions: "Click run to execute the action",
            }],
            overrides: {
                config: routineConfigFixtures.action.simple,
                complexity: 1,
                isComplete: true,
            },
        });
    }

    /**
     * Create a text generation routine version
     */
    async createTextGeneration(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            routine: { id: routineId },
            translations: [{
                language: "en",
                name: "Text Generation Version",
                description: "Generate text using AI",
                instructions: "Provide input text and receive AI-generated content",
            }],
            overrides: {
                config: routineConfigFixtures.generate.withComplexPrompt,
                complexity: 3,
                isComplete: true,
            },
        });
    }

    /**
     * Create a multi-step workflow version
     */
    async createMultiStepWorkflow(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            routine: { id: routineId },
            translations: [{
                language: "en",
                name: "Multi-Step Workflow",
                description: "Complex workflow with multiple steps and branching",
                instructions: "The workflow will guide you through each step automatically",
            }],
            resourceLists: [
                {
                    listFor: "Learn",
                    resources: [
                        {
                            name: "Workflow Guide",
                            link: "https://docs.example.com/workflow-guide",
                            usedFor: "Documentation",
                            translations: [{
                                language: "en",
                                name: "Workflow Guide",
                                description: "Complete guide to understanding this workflow",
                            }],
                        },
                        {
                            name: "Video Tutorial",
                            link: "https://videos.example.com/workflow-tutorial",
                            usedFor: "Tutorial",
                            translations: [{
                                language: "en",
                                name: "Video Tutorial",
                                description: "Step-by-step video walkthrough",
                            }],
                        },
                    ],
                },
                {
                    listFor: "Research",
                    resources: [
                        {
                            name: "Research Paper",
                            link: "https://papers.example.com/workflow-optimization",
                            usedFor: "Research",
                            translations: [{
                                language: "en",
                                name: "Workflow Optimization Research",
                                description: "Academic research on workflow optimization techniques",
                            }],
                        },
                    ],
                },
            ],
            overrides: {
                config: routineConfigFixtures.multiStep.complexWorkflow,
                complexity: 8,
                isComplete: true,
            },
        });
    }

    /**
     * Create a draft version (incomplete)
     */
    async createDraftVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            routine: { id: routineId },
            translations: [{
                language: "en",
                name: "Draft Version",
                description: "Work in progress",
                instructions: "This version is still being developed",
            }],
            overrides: {
                isComplete: false,
                isLatest: true,
                versionLabel: "0.1.0-draft",
                complexity: 1,
            },
        });
    }

    protected async checkModelConstraints(record: Prisma.RoutineVersion): Promise<string[]> {
        const violations: string[] = [];

        // Check version label format
        if (record.versionLabel && !/^\d+\.\d+\.\d+(-\w+)?$/.test(record.versionLabel)) {
            violations.push("Version label must follow semantic versioning format");
        }

        // Check complexity range
        if (record.complexity < 1 || record.complexity > 10) {
            violations.push("Complexity must be between 1 and 10");
        }

        // Check timesCompleted <= timesStarted
        if (record.timesCompleted > record.timesStarted) {
            violations.push("Times completed cannot exceed times started");
        }

        // Check that if isLatest is true, no other version has isLatest for the same routine
        if (record.isLatest && record.routineId) {
            const otherLatest = await this.prisma.routineVersion.count({
                where: {
                    routineId: record.routineId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            if (otherLatest > 0) {
                violations.push("Only one version can be marked as latest");
            }
        }

        // Check config validity
        if (record.config && typeof record.config === "object") {
            const config = record.config as any;
            if (!config.__version) {
                violations.push("Config must have __version field");
            }
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
                timesStarted: -1, // Should be non-negative
                config: "not an object", // Should be object
            },
            invalidVersionLabel: {
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "v1", // Invalid format
                complexity: 5,
                timesStarted: 0,
                timesCompleted: 0,
            },
            invalidComplexity: {
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                complexity: 15, // Out of range (max 10)
                timesStarted: 0,
                timesCompleted: 0,
            },
            invalidCompletionCount: {
                id: generatePK(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionLabel: "1.0.0",
                complexity: 5,
                timesStarted: 5,
                timesCompleted: 10, // More completions than starts
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.RoutineVersionCreateInput> {
        return {
            minimalComplexity: {
                ...this.getMinimalData(),
                complexity: 1,
                config: routineConfigFixtures.minimal,
            },
            maximalComplexity: {
                ...this.getMinimalData(),
                complexity: 10,
                config: routineConfigFixtures.multiStep.complexWorkflow,
            },
            prereleaseVersion: {
                ...this.getMinimalData(),
                versionLabel: "2.0.0-beta.1",
                isComplete: false,
                isLatest: false,
            },
            highUsageVersion: {
                ...this.getMinimalData(),
                timesStarted: 10000,
                timesCompleted: 9500,
                complexity: 3,
            },
            multiLanguageVersion: {
                ...this.getMinimalData(),
                translations: {
                    create: [
                        { id: generatePK(), language: "en", name: "Multi-language Routine", description: "Available in multiple languages" },
                        { id: generatePK(), language: "es", name: "Rutina Multiidioma", description: "Disponible en múltiples idiomas" },
                        { id: generatePK(), language: "fr", name: "Routine Multilingue", description: "Disponible en plusieurs langues" },
                        { id: generatePK(), language: "de", name: "Mehrsprachige Routine", description: "In mehreren Sprachen verfügbar" },
                        { id: generatePK(), language: "ja", name: "多言語ルーチン", description: "複数の言語で利用可能" },
                    ],
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            resourceLists: {
                include: {
                    translations: true,
                    resources: {
                        include: {
                            resourceVersion: {
                                include: {
                                    translations: true,
                                },
                            },
                        },
                    },
                },
            },
            runs: true,
            nodes: {
                include: {
                    translations: true,
                },
            },
            nodeLinks: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.RoutineVersion,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete runs
        if (record.runs?.length) {
            await tx.run.deleteMany({
                where: { routineVersionId: record.id },
            });
        }

        // Delete nodes and node links
        if (record.nodes?.length) {
            // Delete node translations first
            await tx.nodeTranslation.deleteMany({
                where: { nodeId: { in: record.nodes.map((n: any) => n.id) } },
            });
            
            // Delete node links
            if (record.nodeLinks?.length) {
                await tx.nodeLink.deleteMany({
                    where: { routineVersionId: record.id },
                });
            }
            
            // Delete nodes
            await tx.node.deleteMany({
                where: { routineVersionId: record.id },
            });
        }

        // Delete resource lists and their resources
        if (record.resourceLists?.length) {
            for (const list of record.resourceLists) {
                // Delete resource list item translations
                await tx.resourceListTranslation.deleteMany({
                    where: { resourceListId: list.id },
                });
                
                // Delete resources in the list
                if (list.resources?.length) {
                    await tx.resourceListItem.deleteMany({
                        where: { listId: list.id },
                    });
                }
            }
            
            // Delete resource lists
            await tx.resourceList.deleteMany({
                where: { routineVersionId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.routineVersionTranslation.deleteMany({
                where: { routineVersionId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createRoutineVersionDbFactory = (prisma: PrismaClient) => new RoutineVersionDbFactory(prisma);
