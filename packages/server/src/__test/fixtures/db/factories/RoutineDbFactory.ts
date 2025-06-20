import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
import { routineConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface RoutineRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    versions?: Array<{
        versionLabel: string;
        isComplete?: boolean;
        isLatest?: boolean;
        config?: any;
        translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    }>;
    tags?: string[];
    resourceLists?: number;
}

/**
 * Database fixture factory for Routine model
 * Handles routine creation with versions, owners, and configurations
 */
export class RoutineDbFactory extends DatabaseFixtureFactory<
    Prisma.Routine,
    Prisma.RoutineCreateInput,
    Prisma.RoutineInclude,
    Prisma.RoutineUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Routine', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.routine;
    }

    protected getMinimalData(overrides?: Partial<Prisma.RoutineCreateInput>): Prisma.RoutineCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.RoutineCreateInput>): Prisma.RoutineCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            isInternal: false,
            permissions: JSON.stringify({
                canEdit: ["Owner", "Admin"],
                canView: ["Public"],
                canDelete: ["Owner"],
            }),
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionNotes: "Initial version",
                    complexity: 5,
                    timesStarted: 0,
                    timesCompleted: 0,
                    config: routineConfigFixtures.complete,
                    translations: {
                        create: [
                            {
                                id: generatePK(),
                                language: "en",
                                name: "Complete Test Routine",
                                description: "A comprehensive test routine with all features",
                                instructions: "Follow these steps to execute the routine",
                            },
                            {
                                id: generatePK(),
                                language: "es",
                                name: "Rutina de Prueba Completa",
                                description: "Una rutina de prueba integral con todas las funcionalidades",
                                instructions: "Sigue estos pasos para ejecutar la rutina",
                            },
                        ],
                    },
                }],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.RoutineInclude {
        return {
            translations: true,
            versions: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    isComplete: true,
                    isLatest: true,
                    isPrivate: true,
                    translations: true,
                    config: true,
                },
            },
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
            tags: true,
            _count: {
                select: {
                    versions: true,
                    runs: true,
                    forks: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.RoutineCreateInput,
        config: RoutineRelationConfig,
        tx: any
    ): Promise<Prisma.RoutineCreateInput> {
        let data = { ...baseData };

        // Handle owner
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
                create: config.versions.map(version => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: version.isComplete ?? false,
                    isLatest: version.isLatest ?? false,
                    isPrivate: false,
                    versionLabel: version.versionLabel,
                    complexity: 1,
                    timesStarted: 0,
                    timesCompleted: 0,
                    config: version.config ?? routineConfigFixtures.minimal,
                    translations: version.translations ? {
                        create: version.translations.map(trans => ({
                            id: generatePK(),
                            ...trans,
                        })),
                    } : undefined,
                })),
            };
        }

        // Handle tags
        if (config.tags && Array.isArray(config.tags)) {
            data.tags = {
                connectOrCreate: config.tags.map(tag => ({
                    where: { tag },
                    create: {
                        id: generatePK(),
                        tag,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                description: `Tag for ${tag}`,
                            }],
                        },
                    },
                })),
            };
        }

        return data;
    }

    /**
     * Create a simple task routine
     */
    async createSimpleTask(ownerId: string): Promise<Prisma.Routine> {
        return this.createWithRelations({
            owner: { userId: ownerId },
            versions: [{
                versionLabel: "1.0.0",
                isComplete: true,
                isLatest: true,
                config: routineConfigFixtures.action.simple,
                translations: [{
                    language: "en",
                    name: "Simple Task Routine",
                    description: "A basic routine for simple tasks",
                    instructions: "1. Start the task\n2. Complete the action\n3. Review results",
                }],
            }],
            tags: ["task", "simple"],
        });
    }

    /**
     * Create a complex workflow routine
     */
    async createComplexWorkflow(ownerId: string): Promise<Prisma.Routine> {
        return this.createWithRelations({
            owner: { userId: ownerId },
            versions: [{
                versionLabel: "1.0.0",
                isComplete: true,
                isLatest: true,
                config: routineConfigFixtures.multiStep.complexWorkflow,
                translations: [{
                    language: "en",
                    name: "Complex Data Processing Workflow",
                    description: "Multi-step workflow for data processing with branching logic",
                    instructions: "This workflow processes data through multiple stages with conditional routing",
                }],
            }],
            tags: ["workflow", "complex", "data-processing"],
        });
    }

    /**
     * Create an automated routine
     */
    async createAutomatedRoutine(teamId: string): Promise<Prisma.Routine> {
        return this.createWithRelations({
            owner: { teamId },
            versions: [{
                versionLabel: "2.0.0",
                isComplete: true,
                isLatest: true,
                config: routineConfigFixtures.generate.withComplexPrompt,
                translations: [{
                    language: "en",
                    name: "Automated Report Generator",
                    description: "Generates reports automatically based on input data",
                    instructions: "Configure input parameters and the routine will generate customized reports",
                }],
            }],
            tags: ["automation", "reports", "ai-powered"],
        });
    }

    /**
     * Create routine with multiple versions
     */
    async createWithVersionHistory(ownerId: string): Promise<Prisma.Routine> {
        return this.createWithRelations({
            owner: { userId: ownerId },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isComplete: true,
                    isLatest: false,
                    config: routineConfigFixtures.action.simple,
                    translations: [{
                        language: "en",
                        name: "Initial Version",
                        description: "First version of the routine",
                    }],
                },
                {
                    versionLabel: "1.1.0",
                    isComplete: true,
                    isLatest: false,
                    config: routineConfigFixtures.action.withInputMapping,
                    translations: [{
                        language: "en",
                        name: "Enhanced Version",
                        description: "Added input mapping capabilities",
                    }],
                },
                {
                    versionLabel: "2.0.0",
                    isComplete: true,
                    isLatest: true,
                    config: routineConfigFixtures.action.withOutputMapping,
                    translations: [{
                        language: "en",
                        name: "Current Version",
                        description: "Full input/output mapping support",
                    }],
                },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.Routine): Promise<string[]> {
        const violations: string[] = [];

        // Check that routine has at least one version
        const versionCount = await this.prisma.routineVersion.count({
            where: { routineId: record.id },
        });

        if (versionCount === 0) {
            violations.push('Routine must have at least one version');
        }

        // Check that there's exactly one latest version
        const latestVersions = await this.prisma.routineVersion.count({
            where: {
                routineId: record.id,
                isLatest: true,
            },
        });

        if (latestVersions !== 1) {
            violations.push('Routine must have exactly one latest version');
        }

        // Check ownership
        if (!record.ownedByUserId && !record.ownedByTeamId) {
            violations.push('Routine must have an owner (user or team)');
        }

        if (record.ownedByUserId && record.ownedByTeamId) {
            violations.push('Routine cannot be owned by both user and team');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId
                isPrivate: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                isPrivate: "yes", // Should be boolean
                isInternal: 1, // Should be boolean
                permissions: { invalid: "object" }, // Should be string
            },
            bothOwners: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: false,
                ownedByUser: { connect: { id: "user123" } },
                ownedByTeam: { connect: { id: "team123" } }, // Can't have both
            },
            noVersions: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: false,
                // Missing versions
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.RoutineCreateInput> {
        return {
            maxComplexityRoutine: {
                ...this.getMinimalData(),
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isComplete: true,
                        isLatest: true,
                        isPrivate: false,
                        versionLabel: "1.0.0",
                        complexity: 10, // Maximum complexity
                        timesStarted: 0,
                        timesCompleted: 0,
                        config: routineConfigFixtures.multiStep.complexWorkflow,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                name: "Maximum Complexity Routine",
                                description: "A routine with the highest complexity rating",
                            }],
                        },
                    }],
                },
            },
            privateInternalRoutine: {
                ...this.getMinimalData(),
                isPrivate: true,
                isInternal: true,
                permissions: JSON.stringify({
                    canEdit: ["Owner"],
                    canView: ["Owner", "Admin"],
                    canDelete: ["Owner"],
                }),
            },
            minimalConfigRoutine: {
                ...this.getMinimalData(),
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isComplete: false,
                        isLatest: true,
                        isPrivate: false,
                        versionLabel: "0.1.0",
                        complexity: 1,
                        timesStarted: 0,
                        timesCompleted: 0,
                        config: routineConfigFixtures.minimal,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                name: "Minimal Routine",
                                description: "A routine with minimal configuration",
                            }],
                        },
                    }],
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            versions: {
                include: {
                    translations: true,
                    resourceLists: true,
                    runs: true,
                },
            },
            tags: true,
            runs: true,
            forks: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Routine,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete runs
        if (record.runs?.length) {
            await tx.run.deleteMany({
                where: { routineVersionId: { in: record.versions.map((v: any) => v.id) } },
            });
        }

        // Delete forks
        if (record.forks?.length) {
            await tx.routine.updateMany({
                where: { forkedFromId: record.id },
                data: { forkedFromId: null },
            });
        }

        // Delete versions and their translations
        if (record.versions?.length) {
            for (const version of record.versions) {
                if (version.translations?.length) {
                    await tx.routineVersionTranslation.deleteMany({
                        where: { routineVersionId: version.id },
                    });
                }
            }
            await tx.routineVersion.deleteMany({
                where: { routineId: record.id },
            });
        }

        // Delete routine translations
        if (record.translations?.length) {
            await tx.routineTranslation.deleteMany({
                where: { routineId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createRoutineDbFactory = (prisma: PrismaClient) => new RoutineDbFactory(prisma);