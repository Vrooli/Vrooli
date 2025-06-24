import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import { routineConfigFixtures } from "@vrooli/shared/test-fixtures";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface RoutineVersionRelationConfig extends RelationConfig {
    root?: { routineId: string };
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    nodes?: Array<{
        nodeType: string;
        coordinateX?: number;
        coordinateY?: number;
        data?: object;
    }>;
    nodeLinks?: Array<{
        fromNodeId: string;
        toNodeId: string;
        operation?: string;
        whens?: object;
    }>;
}

/**
 * Enhanced database fixture factory for RoutineVersion model
 * Handles versioned routine content with configurations, nodes, and workflow logic
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Version management (latest/complete flags)
 * - Multi-language translations
 * - Routine configuration with various types (action, generate, multi-step)
 * - Node and link management for workflow visualization
 * - Automation capabilities
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class RoutineVersionDbFactory extends EnhancedDatabaseFactory<
    Prisma.RoutineVersionCreateInput,
    Prisma.RoutineVersionCreateInput,
    Prisma.RoutineVersionInclude,
    Prisma.RoutineVersionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("RoutineVersion", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.routineVersion;
    }

    /**
     * Get complete test fixtures for RoutineVersion model
     */
    protected getFixtures(): DbTestFixtures<Prisma.RoutineVersionCreateInput, Prisma.RoutineVersionUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isComplete: false,
                isLatest: true,
                isPrivate: false,
                isAutomatable: true,
                versionLabel: "1.0.0",
                versionIndex: 1,
                routineConfig: routineConfigFixtures.minimal,
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                isAutomatable: true,
                versionLabel: "2.0.0",
                versionIndex: 2,
                versionNotes: "Major update with new automation features",
                complexity: 5,
                resourceIndex: 1,
                routineConfig: routineConfigFixtures.complete,
                translations: {
                    create: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Complete Routine Version",
                            description: "A comprehensive automation routine with all features",
                            instructions: "Follow these steps to execute the routine workflow",
                        },
                        {
                            id: generatePK().toString(),
                            language: "es",
                            name: "Versión Completa de Rutina",
                            description: "Una rutina de automatización integral con todas las características",
                            instructions: "Sigue estos pasos para ejecutar el flujo de trabajo de la rutina",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, versionLabel, versionIndex
                    isComplete: false,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    isComplete: "yes", // Should be boolean
                    isLatest: "true", // Should be boolean
                    isPrivate: 1, // Should be boolean
                    isAutomatable: "false", // Should be boolean
                    versionLabel: 123, // Should be string
                    versionIndex: "one", // Should be number
                    complexity: "five", // Should be number
                    routineConfig: "not an object", // Should be object
                },
                invalidVersionIndex: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                    versionLabel: "1.0.0",
                    versionIndex: -1, // Should be non-negative
                    routineConfig: routineConfigFixtures.minimal,
                },
                invalidConfig: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    routineConfig: { invalid: "config" }, // Invalid config structure
                },
            },
            edgeCases: {
                simpleActionVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    complexity: 1,
                    routineConfig: routineConfigFixtures.action.simple,
                },
                textGenerationVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    complexity: 3,
                    routineConfig: routineConfigFixtures.generate.basic,
                },
                multiStepVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    complexity: 8,
                    routineConfig: routineConfigFixtures.multiStep.sequential,
                },
                manualVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: false, // Not automatable
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    complexity: 4,
                    routineConfig: routineConfigFixtures.minimal,
                },
                betaVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: false,
                    isLatest: false,
                    isPrivate: true,
                    isAutomatable: true,
                    versionLabel: "3.0.0-beta.1",
                    versionIndex: 3,
                    versionNotes: "Beta release - experimental features",
                    complexity: 6,
                    routineConfig: routineConfigFixtures.complete,
                },
                multiLanguageVersion: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    isAutomatable: true,
                    versionLabel: "1.0.0",
                    versionIndex: 1,
                    complexity: 5,
                    routineConfig: routineConfigFixtures.complete,
                    translations: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: generatePK().toString(),
                            language: ["en", "es", "fr", "de", "ja"][i],
                            name: `Routine Version ${i}`,
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
                    versionNotes: "Major update with enhanced automation",
                    complexity: 7,
                    routineConfig: routineConfigFixtures.multiStep.complexWorkflow,
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

    protected generateMinimalData(overrides?: Partial<Prisma.RoutineVersionCreateInput>): Prisma.RoutineVersionCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isComplete: false,
            isLatest: true,
            isPrivate: false,
            isAutomatable: true,
            versionLabel: "1.0.0",
            versionIndex: 1,
            routineConfig: routineConfigFixtures.minimal,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.RoutineVersionCreateInput>): Prisma.RoutineVersionCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isComplete: true,
            isLatest: true,
            isPrivate: false,
            isAutomatable: true,
            versionLabel: "2.0.0",
            versionIndex: 2,
            versionNotes: "Major update with new automation features",
            complexity: 5,
            resourceIndex: 1,
            routineConfig: routineConfigFixtures.complete,
            translations: {
                create: [
                    {
                        id: generatePK().toString(),
                        language: "en",
                        name: "Complete Routine Version",
                        description: "A comprehensive automation routine with all features",
                        instructions: "Follow these steps to execute the routine workflow",
                    },
                    {
                        id: generatePK().toString(),
                        language: "es",
                        name: "Versión Completa de Rutina",
                        description: "Una rutina de automatización integral con todas las características",
                        instructions: "Sigue estos pasos para ejecutar el flujo de trabajo de la rutina",
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
            simpleActionVersion: {
                name: "simpleActionVersion",
                description: "Simple action routine version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        complexity: 1,
                        routineConfig: routineConfigFixtures.action.simple,
                    },
                    translations: [{
                        language: "en",
                        name: "Simple Action",
                        description: "Executes a single automated action",
                        instructions: "Click run to execute",
                    }],
                },
            },
            textGenerationVersion: {
                name: "textGenerationVersion",
                description: "AI text generation routine version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        complexity: 3,
                        routineConfig: routineConfigFixtures.generate.basic,
                    },
                    translations: [{
                        language: "en",
                        name: "Text Generator",
                        description: "AI-powered text generation",
                        instructions: "Provide input prompt and configure parameters",
                    }],
                },
            },
            complexWorkflowVersion: {
                name: "complexWorkflowVersion",
                description: "Complex multi-step workflow version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        complexity: 9,
                        routineConfig: routineConfigFixtures.multiStep.complexWorkflow,
                    },
                    translations: [{
                        language: "en",
                        name: "Complex Workflow",
                        description: "Advanced multi-step automation workflow",
                        instructions: "Review all steps before execution",
                    }],
                    nodes: [
                        {
                            nodeType: "start",
                            coordinateX: 100,
                            coordinateY: 100,
                        },
                        {
                            nodeType: "action",
                            coordinateX: 200,
                            coordinateY: 200,
                            data: { action: "fetchData" },
                        },
                        {
                            nodeType: "decision",
                            coordinateX: 300,
                            coordinateY: 300,
                            data: { condition: "dataExists" },
                        },
                        {
                            nodeType: "end",
                            coordinateX: 400,
                            coordinateY: 400,
                        },
                    ],
                },
            },
            dataTransformationVersion: {
                name: "dataTransformationVersion",
                description: "Data transformation routine version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        complexity: 6,
                        routineConfig: routineConfigFixtures.variants.dataTransformation,
                    },
                    translations: [{
                        language: "en",
                        name: "Data Transformer",
                        description: "Transforms data through multiple stages",
                        instructions: "Upload data and configure transformation rules",
                    }],
                },
            },
            manualProcessVersion: {
                name: "manualProcessVersion",
                description: "Manual process routine version",
                config: {
                    overrides: {
                        isComplete: true,
                        isLatest: true,
                        isAutomatable: false,
                        complexity: 4,
                        simplicity: 6,
                        routineConfig: routineConfigFixtures.minimal,
                    },
                    translations: [{
                        language: "en",
                        name: "Manual Process",
                        description: "Step-by-step manual workflow",
                        instructions: "Complete each step manually and mark as done",
                    }],
                },
            },
            betaVersion: {
                name: "betaVersion",
                description: "Beta/experimental routine version",
                config: {
                    overrides: {
                        isComplete: false,
                        isLatest: false,
                        isPrivate: true,
                        versionLabel: "3.0.0-beta.1",
                        versionIndex: 3,
                        versionNotes: "Beta release - experimental features",
                        complexity: 6,
                        routineConfig: routineConfigFixtures.complete,
                    },
                    translations: [{
                        language: "en",
                        name: "Beta Version",
                        description: "Experimental features for testing",
                        instructions: "This is a beta version. Use with caution.",
                    }],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.RoutineVersionInclude {
        return {
            root: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                    isPrivate: true,
                },
            },
            translations: true,
            nodes: {
                select: {
                    id: true,
                    nodeType: true,
                    coordinateX: true,
                    coordinateY: true,
                    data: true,
                },
                orderBy: {
                    coordinateX: "asc",
                },
            },
            nodeLinks: {
                select: {
                    id: true,
                    operation: true,
                    whens: true,
                    from: {
                        select: {
                            id: true,
                            nodeType: true,
                        },
                    },
                    to: {
                        select: {
                            id: true,
                            nodeType: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    translations: true,
                    nodes: true,
                    nodeLinks: true,
                    runs: true,
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

        // Handle root routine connection
        if (config.root?.routineId) {
            data.root = { connect: { id: config.root.routineId } };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK().toString(),
                    ...trans,
                })),
            };
        }

        // Handle nodes (workflow visualization)
        if (config.nodes && Array.isArray(config.nodes)) {
            data.nodes = {
                create: config.nodes.map(node => ({
                    id: generatePK().toString(),
                    nodeType: node.nodeType,
                    coordinateX: node.coordinateX ?? 0,
                    coordinateY: node.coordinateY ?? 0,
                    data: node.data ?? {},
                })),
            };
        }

        // Handle node links (workflow connections)
        if (config.nodeLinks && Array.isArray(config.nodeLinks)) {
            data.nodeLinks = {
                create: config.nodeLinks.map(link => ({
                    id: generatePK().toString(),
                    from: { connect: { id: link.fromNodeId } },
                    to: { connect: { id: link.toNodeId } },
                    operation: link.operation,
                    whens: link.whens,
                })),
            };
        }

        return data;
    }

    /**
     * Create a simple action version
     */
    async createSimpleActionVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            root: { routineId },
            overrides: {
                isComplete: true,
                isLatest: true,
                complexity: 1,
                routineConfig: routineConfigFixtures.action.simple,
            },
            translations: [{
                language: "en",
                name: "Simple Action",
                description: "Executes a single automated action",
                instructions: "Click run to execute",
            }],
        });
    }

    /**
     * Create a text generation version
     */
    async createTextGenerationVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            root: { routineId },
            overrides: {
                isComplete: true,
                isLatest: true,
                complexity: 3,
                routineConfig: routineConfigFixtures.generate.basic,
            },
            translations: [{
                language: "en",
                name: "Text Generator",
                description: "AI-powered text generation",
                instructions: "Provide input prompt and configure parameters",
            }],
        });
    }

    /**
     * Create a complex workflow version
     */
    async createComplexWorkflowVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.seedScenario("complexWorkflowVersion");
    }

    /**
     * Create a data transformation version
     */
    async createDataTransformationVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            root: { routineId },
            overrides: {
                isComplete: true,
                isLatest: true,
                complexity: 6,
                routineConfig: routineConfigFixtures.variants.dataTransformation,
            },
            translations: [{
                language: "en",
                name: "Data Transformer",
                description: "Transforms data through multiple stages",
                instructions: "Upload data and configure transformation rules",
            }],
        });
    }

    /**
     * Create a manual process version
     */
    async createManualProcessVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.seedScenario("manualProcessVersion");
    }

    /**
     * Create a beta version
     */
    async createBetaVersion(routineId: string): Promise<Prisma.RoutineVersion> {
        return this.createWithRelations({
            root: { routineId },
            overrides: {
                isComplete: false,
                isLatest: false,
                isPrivate: true,
                versionLabel: "3.0.0-beta.1",
                versionIndex: 3,
                versionNotes: "Beta release - experimental features",
                complexity: 6,
                routineConfig: routineConfigFixtures.complete,
            },
            translations: [{
                language: "en",
                name: "Beta Version",
                description: "Experimental features for testing",
                instructions: "This is a beta version. Use with caution.",
            }],
        });
    }

    /**
     * Create version with workflow nodes and links
     */
    async createVersionWithWorkflow(routineId: string): Promise<Prisma.RoutineVersion> {
        const nodes = [
            {
                nodeType: "start",
                coordinateX: 100,
                coordinateY: 100,
            },
            {
                nodeType: "action",
                coordinateX: 200,
                coordinateY: 200,
                data: { action: "processData" },
            },
            {
                nodeType: "end",
                coordinateX: 300,
                coordinateY: 300,
            },
        ];

        const version = await this.createWithRelations({
            root: { routineId },
            overrides: {
                isComplete: true,
                isLatest: true,
                complexity: 5,
                routineConfig: routineConfigFixtures.multiStep.sequential,
            },
            translations: [{
                language: "en",
                name: "Workflow with Nodes",
                description: "Routine with visual workflow representation",
            }],
            nodes,
        });

        // Create links after nodes are created
        const createdNodes = await this.prisma.routineVersionNode.findMany({
            where: { routineVersionId: version.id },
            orderBy: { coordinateX: "asc" },
        });

        if (createdNodes.length >= 3) {
            await this.prisma.routineVersionNodeLink.createMany({
                data: [
                    {
                        id: generatePK().toString(),
                        routineVersionId: version.id,
                        fromId: createdNodes[0].id,
                        toId: createdNodes[1].id,
                        operation: "next",
                    },
                    {
                        id: generatePK().toString(),
                        routineVersionId: version.id,
                        fromId: createdNodes[1].id,
                        toId: createdNodes[2].id,
                        operation: "complete",
                    },
                ],
            });
        }

        return version;
    }

    protected async checkModelConstraints(record: Prisma.RoutineVersion): Promise<string[]> {
        const violations: string[] = [];

        // Check complexity range
        if (record.complexity !== null && (record.complexity < 1 || record.complexity > 10)) {
            violations.push("Complexity must be between 1 and 10");
        }

        // Check simplicity range
        if (record.simplicity !== null && (record.simplicity < 1 || record.simplicity > 10)) {
            violations.push("Simplicity must be between 1 and 10");
        }

        // Check version index is non-negative
        if (record.versionIndex < 0) {
            violations.push("Version index must be non-negative");
        }

        // Check that version belongs to a routine
        if (!record.rootId) {
            violations.push("RoutineVersion must belong to a Routine");
        }

        // Check version label format
        if (record.versionLabel && !/^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$/.test(record.versionLabel)) {
            violations.push("Version label should follow semantic versioning format");
        }

        // Check that only one version is marked as latest per routine
        if (record.isLatest && record.rootId) {
            const otherLatest = await this.prisma.routineVersion.count({
                where: {
                    rootId: record.rootId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            if (otherLatest > 0) {
                violations.push("Only one version can be marked as latest per routine");
            }
        }

        // Check routineConfig structure
        if (record.routineConfig && 
            (!record.routineConfig.__version || 
             typeof record.routineConfig.__version !== "string")) {
            violations.push("Routine config must have a valid __version field");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            nodes: true,
            nodeLinks: true,
            runs: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.RoutineVersion,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete runs
        if (shouldDelete("runs") && record.runs?.length) {
            await tx.run.deleteMany({
                where: { routineVersionId: record.id },
            });
        }

        // Delete node links
        if (shouldDelete("nodeLinks") && record.nodeLinks?.length) {
            await tx.routineVersionNodeLink.deleteMany({
                where: { routineVersionId: record.id },
            });
        }

        // Delete nodes
        if (shouldDelete("nodes") && record.nodes?.length) {
            await tx.routineVersionNode.deleteMany({
                where: { routineVersionId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete("translations") && record.translations?.length) {
            await tx.routineVersionTranslation.deleteMany({
                where: { routineVersionId: record.id },
            });
        }
    }

    /**
     * Create version history for testing
     */
    async createVersionHistory(routineId: string, count = 3): Promise<Prisma.RoutineVersion[]> {
        const versions: Prisma.RoutineVersion[] = [];
        const configs = [
            routineConfigFixtures.action.simple,
            routineConfigFixtures.generate.basic,
            routineConfigFixtures.multiStep.sequential,
        ];
        
        for (let i = 0; i < count; i++) {
            const isLatest = i === count - 1;
            const version = await this.createWithRelations({
                root: { routineId },
                overrides: {
                    versionLabel: `${i + 1}.0.0`,
                    versionIndex: i + 1,
                    isLatest,
                    isComplete: true,
                    versionNotes: isLatest ? "Latest version" : `Version ${i + 1} release notes`,
                    complexity: Math.min(10, i + 3),
                    routineConfig: configs[i % configs.length],
                },
                translations: [{
                    language: "en",
                    name: `Version ${i + 1}.0.0`,
                    description: isLatest ? "Latest version with all features" : `Version ${i + 1} of the routine`,
                }],
            });
            versions.push(version);
        }
        
        return versions;
    }
}

// Export factory creator function
export const createRoutineVersionDbFactory = (prisma: PrismaClient) => 
    RoutineVersionDbFactory.getInstance("RoutineVersion", prisma);

// Export the class for type usage
export { RoutineVersionDbFactory as RoutineVersionDbFactoryClass };
