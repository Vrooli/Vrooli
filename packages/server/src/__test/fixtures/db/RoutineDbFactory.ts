import { generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface RoutineRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    versions?: Array<{
        versionLabel: string;
        isLatest?: boolean;
        isComplete?: boolean;
        isPrivate?: boolean;
        isAutomatable?: boolean;
        translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    }>;
    tags?: string[];
    translations?: Array<{ language: string; name: string; description?: string }>;
}

/**
 * Enhanced database fixture factory for Routine model
 * Handles routines (workflows/automations) with versions, translations, and configurations
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Version management with RoutineVersion
 * - Owner relationships (user or team)
 * - Tag associations
 * - Multi-language support
 * - Automation capabilities
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class RoutineDbFactory extends EnhancedDatabaseFactory<
    routine,
    Prisma.routineCreateInput,
    Prisma.routineInclude,
    Prisma.routineUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("Routine", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.routine;
    }

    /**
     * Get complete test fixtures for Routine model
     */
    protected getFixtures(): DbTestFixtures<Prisma.routineCreateInput, Prisma.routineUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: generatePublicId(),
                handle: `routine_${nanoid()}`,
                isPrivate: false,
                isInternal: false,
            },
            complete: {
                id: this.generateId(),
                publicId: generatePublicId(),
                handle: `complete_routine_${nanoid()}`,
                isPrivate: false,
                isInternal: false,
                permissions: JSON.stringify({
                    canEdit: ["Owner", "Admin"],
                    canView: ["Public"],
                    canDelete: ["Owner"],
                    canRun: ["Public"],
                }),
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            name: "Complete Test Routine",
                            description: "A comprehensive automation routine with all features",
                            instructions: "Follow these steps to execute the routine",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            name: "Rutina de Prueba Completa",
                            description: "Una rutina de automatización integral con todas las características",
                            instructions: "Sigue estos pasos para ejecutar la rutina",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, handle
                    isPrivate: false,
                    isInternal: false,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    handle: true, // Should be string
                    isPrivate: "yes", // Should be boolean
                    isInternal: 1, // Should be boolean
                    permissions: { invalid: "object" }, // Should be string
                },
                duplicateHandle: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: "existing_routine_handle", // Assumes this exists
                    isPrivate: false,
                    isInternal: false,
                },
                bothOwners: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `routine_${nanoid()}`,
                    isPrivate: false,
                    isInternal: false,
                    ownedByUser: { connect: { id: "user123" } },
                    ownedByTeam: { connect: { id: "team123" } }, // Can't have both
                },
            },
            edgeCases: {
                maxLengthHandle: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: "rout_" + "a".repeat(45), // Max length handle
                    isPrivate: false,
                    isInternal: false,
                },
                unicodeNameRoutine: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `unicode_routine_${nanoid()}`,
                    isPrivate: false,
                    isInternal: false,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            name: "ルーチン 🤖", // Unicode characters
                            description: "Unicode routine name",
                        }],
                    },
                },
                privateInternalRoutine: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `private_routine_${nanoid()}`,
                    isPrivate: true,
                    isInternal: true,
                    permissions: JSON.stringify({
                        canEdit: ["Owner"],
                        canView: ["TeamMember"],
                        canDelete: ["Owner"],
                        canRun: ["TeamMember"],
                    }),
                },
                publicAutomationRoutine: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `automation_${nanoid()}`,
                    isPrivate: false,
                    isInternal: false,
                    permissions: JSON.stringify({
                        canEdit: ["Owner", "Admin"],
                        canView: ["Public"],
                        canDelete: ["Owner"],
                        canRun: ["Public"],
                    }),
                },
                multiLanguageRoutine: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `multilang_routine_${nanoid()}`,
                    isPrivate: false,
                    isInternal: false,
                    translations: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: this.generateId(),
                            language: ["en", "es", "fr", "de", "ja"][i],
                            name: `Routine Name ${i}`,
                            description: `Routine description in language ${i}`,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    isPrivate: true,
                },
                complete: {
                    isPrivate: false,
                    isInternal: true,
                    permissions: JSON.stringify({
                        canEdit: ["Owner", "Admin", "Contributor"],
                        canView: ["TeamMember"],
                        canDelete: ["Owner"],
                        canRun: ["TeamMember", "Guest"],
                    }),
                    translations: {
                        update: [{
                            where: { id: "translation_id" },
                            data: { 
                                description: "Updated routine description",
                                instructions: "Updated execution instructions",
                            },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.routineCreateInput>): Prisma.routineCreateInput {
        const uniqueHandle = `routine_${nanoid()}`;
        
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isPrivate: false,
            isInternal: false,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.routineCreateInput>): Prisma.routineCreateInput {
        const uniqueHandle = `complete_routine_${nanoid()}`;
        
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isPrivate: false,
            isInternal: false,
            permissions: JSON.stringify({
                canEdit: ["Owner", "Admin"],
                canView: ["Public"],
                canDelete: ["Owner"],
                canRun: ["Public"],
            }),
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        name: "Complete Test Routine",
                        description: "A comprehensive automation routine with all features",
                        instructions: "Follow these steps to execute the routine",
                    },
                    {
                        id: this.generateId(),
                        language: "es",
                        name: "Rutina de Prueba Completa",
                        description: "Una rutina de automatización integral con todas las características",
                        instructions: "Sigue estos pasos para ejecutar la rutina",
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
            simpleActionRoutine: {
                name: "simpleActionRoutine",
                description: "Simple action routine for basic automation",
                config: {
                    overrides: {
                        handle: `action_routine_${nanoid()}`,
                        isPrivate: false,
                    },
                    versions: [{
                        versionLabel: "1.0.0",
                        isLatest: true,
                        isComplete: true,
                        isAutomatable: true,
                        translations: [{
                            language: "en",
                            name: "Simple Action Routine",
                            description: "Performs a single automated action",
                            instructions: "Click run to execute the action",
                        }],
                    }],
                    tags: ["automation", "action", "simple"],
                },
            },
            textGenerationRoutine: {
                name: "textGenerationRoutine",
                description: "AI text generation routine",
                config: {
                    overrides: {
                        handle: `text_gen_${nanoid()}`,
                        isPrivate: false,
                    },
                    versions: [{
                        versionLabel: "1.0.0",
                        isLatest: true,
                        isComplete: true,
                        isAutomatable: true,
                        translations: [{
                            language: "en",
                            name: "Text Generation Routine",
                            description: "Generates text using AI models",
                            instructions: "Provide input text and select generation parameters",
                        }],
                    }],
                    tags: ["ai", "generation", "text"],
                },
            },
            multiStepWorkflow: {
                name: "multiStepWorkflow",
                description: "Complex multi-step workflow routine",
                config: {
                    overrides: {
                        handle: `workflow_${nanoid()}`,
                        isPrivate: false,
                    },
                    versions: [{
                        versionLabel: "2.0.0",
                        isLatest: true,
                        isComplete: true,
                        isAutomatable: true,
                        translations: [{
                            language: "en",
                            name: "Multi-Step Workflow",
                            description: "Complex workflow with multiple sequential and parallel steps",
                            instructions: "Follow the step-by-step guide through the workflow",
                        }],
                    }],
                    tags: ["workflow", "complex", "automation"],
                },
            },
            dataTransformationRoutine: {
                name: "dataTransformationRoutine",
                description: "Data transformation and processing routine",
                config: {
                    overrides: {
                        handle: `data_transform_${nanoid()}`,
                        isPrivate: false,
                    },
                    versions: [{
                        versionLabel: "1.0.0",
                        isLatest: true,
                        isComplete: true,
                        isAutomatable: true,
                        translations: [{
                            language: "en",
                            name: "Data Transformation Pipeline",
                            description: "Transforms and processes data through multiple stages",
                            instructions: "Upload your data and configure transformation parameters",
                        }],
                    }],
                    tags: ["data", "transformation", "pipeline"],
                },
            },
            manualProcessRoutine: {
                name: "manualProcessRoutine",
                description: "Manual process routine requiring human interaction",
                config: {
                    overrides: {
                        handle: `manual_process_${nanoid()}`,
                        isPrivate: false,
                    },
                    versions: [{
                        versionLabel: "1.0.0",
                        isLatest: true,
                        isComplete: true,
                        isAutomatable: false, // Not automatable
                        translations: [{
                            language: "en",
                            name: "Manual Review Process",
                            description: "Step-by-step manual review and approval process",
                            instructions: "Complete each step manually and mark as done",
                        }],
                    }],
                    tags: ["manual", "process", "review"],
                },
            },
            privateTeamRoutine: {
                name: "privateTeamRoutine",
                description: "Private routine for team use only",
                config: {
                    overrides: {
                        handle: `team_routine_${nanoid()}`,
                        isPrivate: true,
                        isInternal: true,
                        permissions: JSON.stringify({
                            canEdit: ["TeamOwner", "TeamAdmin"],
                            canView: ["TeamMember"],
                            canDelete: ["TeamOwner"],
                            canRun: ["TeamMember"],
                        }),
                    },
                    versions: [{
                        versionLabel: "1.0.0",
                        isLatest: true,
                        isComplete: true,
                        isAutomatable: true,
                        translations: [{
                            language: "en",
                            name: "Internal Team Process",
                            description: "Confidential routine for team operations",
                            instructions: "This routine is for internal team use only",
                        }],
                    }],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.routineInclude {
        return {
            translations: true,
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
            versions: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    isLatest: true,
                    isComplete: true,
                    isPrivate: true,
                    isAutomatable: true,
                    createdAt: true,
                },
                orderBy: {
                    versionIndex: "desc",
                },
            },
            tags: {
                select: {
                    tag: {
                        select: {
                            id: true,
                            tag: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    versions: true,
                    bookmarks: true,
                    runs: true,
                    views: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.routineCreateInput,
        config: RoutineRelationConfig,
        tx: any,
    ): Promise<Prisma.routineCreateInput> {
        const data = { ...baseData };

        // Handle owner (user or team)
        if (config.owner) {
            if (config.owner.userId) {
                data.ownedByUser = { connect: { id: config.owner.userId } };
            } else if (config.owner.teamId) {
                data.ownedByTeam = { connect: { id: config.owner.teamId } };
            }
        }

        // Handle versions
        if (config.versions && Array.isArray(config.versions)) {
            data.versions = {
                create: config.versions.map((version, index) => ({
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    versionLabel: version.versionLabel,
                    versionIndex: index + 1,
                    isLatest: version.isLatest ?? (index === config.versions!.length - 1),
                    isComplete: version.isComplete ?? true,
                    isPrivate: version.isPrivate ?? false,
                    isAutomatable: version.isAutomatable ?? true,
                    translations: version.translations ? {
                        create: version.translations.map(trans => ({
                            id: this.generateId(),
                            ...trans,
                        })),
                    } : undefined,
                })),
            };
        }

        // Handle tags
        if (config.tags && Array.isArray(config.tags)) {
            data.tags = {
                create: config.tags.map(tagName => ({
                    id: this.generateId(),
                    tag: {
                        connectOrCreate: {
                            where: { tag: tagName },
                            create: {
                                id: this.generateId(),
                                tag: tagName,
                                translations: {
                                    create: [{
                                        id: this.generateId(),
                                        language: "en",
                                        description: `Tag for ${tagName}`,
                                    }],
                                },
                            },
                        },
                    },
                })),
            };
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

        return data;
    }

    /**
     * Create a simple action routine
     */
    async createSimpleActionRoutine(): Promise<Prisma.Routine> {
        return this.seedScenario("simpleActionRoutine");
    }

    /**
     * Create a text generation routine
     */
    async createTextGenerationRoutine(): Promise<Prisma.Routine> {
        return this.seedScenario("textGenerationRoutine");
    }

    /**
     * Create a multi-step workflow
     */
    async createMultiStepWorkflow(): Promise<Prisma.Routine> {
        return this.seedScenario("multiStepWorkflow");
    }

    /**
     * Create a data transformation routine
     */
    async createDataTransformationRoutine(): Promise<Prisma.Routine> {
        return this.seedScenario("dataTransformationRoutine");
    }

    /**
     * Create a manual process routine
     */
    async createManualProcessRoutine(): Promise<Prisma.Routine> {
        return this.seedScenario("manualProcessRoutine");
    }

    /**
     * Create a private team routine
     */
    async createPrivateTeamRoutine(teamId: string): Promise<Prisma.Routine> {
        return this.createWithRelations({
            owner: { teamId },
            overrides: {
                handle: `team_routine_${nanoid()}`,
                isPrivate: true,
                isInternal: true,
                permissions: JSON.stringify({
                    canEdit: ["TeamOwner", "TeamAdmin"],
                    canView: ["TeamMember"],
                    canDelete: ["TeamOwner"],
                    canRun: ["TeamMember"],
                }),
            },
            versions: [{
                versionLabel: "1.0.0",
                isLatest: true,
                isComplete: true,
                isAutomatable: true,
                translations: [{
                    language: "en",
                    name: "Internal Team Process",
                    description: "Confidential routine for team operations",
                    instructions: "This routine is for internal team use only",
                }],
            }],
        });
    }

    /**
     * Create routine with version history
     */
    async createRoutineWithVersionHistory(): Promise<Prisma.Routine> {
        return this.createWithRelations({
            overrides: {
                handle: `versioned_routine_${nanoid()}`,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isComplete: true,
                    isLatest: false,
                    translations: [{
                        language: "en",
                        name: "Routine v1.0.0",
                        description: "Initial release",
                    }],
                },
                {
                    versionLabel: "1.1.0",
                    isComplete: true,
                    isLatest: false,
                    translations: [{
                        language: "en",
                        name: "Routine v1.1.0",
                        description: "Bug fixes and improvements",
                    }],
                },
                {
                    versionLabel: "2.0.0",
                    isComplete: true,
                    isLatest: true,
                    translations: [{
                        language: "en",
                        name: "Routine v2.0.0",
                        description: "Major update with new features",
                    }],
                },
            ],
            tags: ["versioned", "stable", "production"],
        });
    }

    protected async checkModelConstraints(record: Prisma.Routine): Promise<string[]> {
        const violations: string[] = [];
        
        // Check handle uniqueness
        if (record.handle) {
            const duplicate = await this.prisma.routine.findFirst({
                where: { 
                    handle: record.handle,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push("Handle must be unique");
            }
        }

        // Check handle format
        if (record.handle && !/^[a-zA-Z0-9_]+$/.test(record.handle)) {
            violations.push("Handle contains invalid characters");
        }

        // Check that routine has at least one version
        const versions = await this.prisma.routine_version.count({
            where: { rootId: record.id },
        });
        
        if (versions === 0) {
            violations.push("Routine must have at least one version");
        }

        // Check that only one version is marked as latest
        const latestVersions = await this.prisma.routine_version.count({
            where: {
                rootId: record.id,
                isLatest: true,
            },
        });
        
        if (latestVersions > 1) {
            violations.push("Routine can only have one latest version");
        }

        // Check ownership
        if (!record.ownedByUserId && !record.ownedByTeamId) {
            violations.push("Routine must have an owner (user or team)");
        }

        if (record.ownedByUserId && record.ownedByTeamId) {
            violations.push("Routine cannot be owned by both user and team");
        }

        // Check private routine has owner
        if (record.isPrivate && !record.ownedByUserId && !record.ownedByTeamId) {
            violations.push("Private routine must have an owner");
        }

        // Check internal routines are team-owned
        if (record.isInternal && !record.ownedByTeamId) {
            violations.push("Internal routines should be team-owned");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            versions: {
                include: {
                    translations: true,
                    nodes: true,
                    nodeLinks: true,
                },
            },
            translations: true,
            tags: true,
            bookmarks: true,
            runs: true,
            views: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Routine,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);
        
        // Delete in order of dependencies
        
        // Delete runs
        if (shouldDelete("runs") && record.runs?.length) {
            await tx.run.deleteMany({
                where: { routineId: record.id },
            });
        }

        // Delete routine versions (cascade will handle their related records)
        if (shouldDelete("versions") && record.versions?.length) {
            for (const version of record.versions) {
                // Delete version translations
                if (version.translations?.length) {
                    await tx.routine_version_translation.deleteMany({
                        where: { routineVersionId: version.id },
                    });
                }
                
                // Delete nodes and links
                if (version.nodes?.length) {
                    await tx.routineVersionNode.deleteMany({
                        where: { routineVersionId: version.id },
                    });
                }
                
                if (version.nodeLinks?.length) {
                    await tx.routineVersionNodeLink.deleteMany({
                        where: { routineVersionId: version.id },
                    });
                }
            }
            
            await tx.routine_version.deleteMany({
                where: { rootId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete("bookmarks") && record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { routineId: record.id },
            });
        }

        // Delete views
        if (shouldDelete("views") && record.views?.length) {
            await tx.view.deleteMany({
                where: { routineId: record.id },
            });
        }

        // Delete tag relationships
        if (shouldDelete("tags") && record.tags?.length) {
            await tx.routineTag.deleteMany({
                where: { routineId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete("translations") && record.translations?.length) {
            await tx.routine_translation.deleteMany({
                where: { routineId: record.id },
            });
        }
    }

    /**
     * Create routine collection for testing
     */
    async createRoutineCollection(ownerId: string, count = 3): Promise<Prisma.Routine[]> {
        const routines: Prisma.Routine[] = [];
        const types = ["simpleActionRoutine", "textGenerationRoutine", "multiStepWorkflow", "dataTransformationRoutine"];
        
        for (let i = 0; i < count; i++) {
            const type = types[i % types.length];
            const routine = await this.seedScenario(type as any);
            routines.push(routine as unknown as Prisma.Routine);
        }
        
        return routines;
    }
}

// Export factory creator function
export const createRoutineDbFactory = (prisma: PrismaClient) => 
    new RoutineDbFactory(prisma);

// Export the class for type usage
export { RoutineDbFactory as RoutineDbFactoryClass };
