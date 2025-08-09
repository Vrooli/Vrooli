/* eslint-disable no-magic-numbers */
import { type PrismaClient } from "@prisma/client";
import { generatePublicId } from "@vrooli/shared";
import { routineConfigFixtures } from "@vrooli/shared/test-fixtures/config";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface RoutineVersionRelationConfig extends RelationConfig {
    root?: { routineId: bigint };
    translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    // Nodes and node links are now stored in the config.graph JSONB field
    // No separate database relations needed
}

/**
 * Enhanced database fixture factory for RoutineVersion model
 * Handles versioned routine content with configurations and workflow logic
 * 
 * MIGRATION STATUS: Updated to use resource_version model and config-based graph system
 * - Routine versions are stored as resource_version with resourceType: ResourceType.Routine
 * - Graph data (nodes/links) is now stored in config.graph JSONB field, not separate tables
 * - Supports BPMN-2.0 and Sequential graph types via config
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Version management (latest/complete flags)
 * - Multi-language translations
 * - Routine configuration with various types (action, generate, multi-step)
 * - Config-based workflow definitions (BPMN/Sequential)
 * - Automation capabilities
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class RoutineVersionDbFactory extends EnhancedDatabaseFactory<
    any, // TODO: Should be 'resource_version' when migrated
    any, // TODO: Should be Prisma.resource_versionCreateInput when migrated
    any, // TODO: Should be Prisma.resource_versionInclude when migrated
    any  // TODO: Should be Prisma.resource_versionUpdateInput when migrated
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("RoutineVersion", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        // Updated to use resource_version after migration
        return this.prisma.resource_version;
    }

    /**
     * Get complete test fixtures for RoutineVersion model
     */
    protected getFixtures(): DbTestFixtures<any, any> {
        return {
            minimal: {
                id: this.generateId(),
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
                id: this.generateId(),
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
                            id: this.generateId(),
                            language: "en",
                            name: "Complete Routine Version",
                            description: "A comprehensive automation routine with all features",
                            instructions: "Follow these steps to execute the routine workflow",
                        },
                        {
                            id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                    id: this.generateId(),
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
                            id: this.generateId(),
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

    protected generateMinimalData(overrides?: Partial<any>): any {
        return {
            id: this.generateId(),
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

    protected generateCompleteData(overrides?: Partial<any>): any {
        return {
            id: this.generateId(),
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
                        id: this.generateId(),
                        language: "en",
                        name: "Complete Routine Version",
                        description: "A comprehensive automation routine with all features",
                        instructions: "Follow these steps to execute the routine workflow",
                    },
                    {
                        id: this.generateId(),
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
                    // Graph data is now defined in routineConfig.graph field using BPMN or Sequential format
                    // No separate nodes array needed - workflow structure is in the config
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

    protected getDefaultInclude(): any {
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
            // Graph data (nodes/nodeLinks) is now stored in the config JSONB field
            // No separate database relations needed
            _count: {
                select: {
                    translations: true,
                    runs: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: any,
        config: RoutineVersionRelationConfig,
        tx: any,
    ): Promise<any> {
        const data = { ...baseData };

        // Handle root routine connection
        if (config.root?.routineId) {
            data.root = { connect: { id: config.root.routineId } };
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

        // Graph data (nodes/nodeLinks) is now stored in the routineConfig.graph field
        // No separate database entities need to be created

        return data;
    }

    /**
     * Create a simple action version
     */
    async createSimpleActionVersion(routineId: bigint): Promise<any> {
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
    async createTextGenerationVersion(routineId: bigint): Promise<any> {
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
    async createComplexWorkflowVersion(routineId: bigint): Promise<any> {
        return this.seedScenario("complexWorkflowVersion");
    }

    /**
     * Create a data transformation version
     */
    async createDataTransformationVersion(routineId: bigint): Promise<any> {
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
    async createManualProcessVersion(routineId: bigint): Promise<any> {
        return this.seedScenario("manualProcessVersion");
    }

    /**
     * Create a beta version
     */
    async createBetaVersion(routineId: bigint): Promise<any> {
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
     * Create version with config-based workflow (using Sequential graph type)
     */
    async createVersionWithWorkflow(routineId: bigint): Promise<any> {
        // Use Sequential graph configuration instead of separate node/nodeLink tables
        const workflowConfig = {
            ...routineConfigFixtures.multiStep.sequential,
            graph: {
                __version: "1.0",
                __type: "Sequential" as const,
                schema: {
                    steps: [
                        {
                            id: "step1",
                            name: "Process Data",
                            description: "Main data processing step",
                            subroutineId: "sub_process_data",
                            inputMap: { "data": "inputData" },
                            outputMap: { "result": "processedData" },
                        },
                    ],
                    rootContext: {
                        inputMap: { "inputData": "workflowInput" },
                        outputMap: { "workflowOutput": "processedData" },
                    },
                    executionMode: "sequential" as const,
                },
            },
        };

        const version = await this.createWithRelations({
            root: { routineId },
            overrides: {
                isComplete: true,
                isLatest: true,
                complexity: 5,
                routineConfig: workflowConfig,
            },
            translations: [{
                language: "en",
                name: "Workflow with Config-based Graph",
                description: "Routine with config-based workflow representation",
            }],
        });

        // No need to create separate node/nodeLink records - workflow is now stored in config
        return version;
    }

    protected async checkModelConstraints(record: any): Promise<string[]> {
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
            // Updated to use resource_version after migration
            const otherLatest = await this.prisma.resource_version.count({
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
            runs: true,
            // Graph data (nodes/nodeLinks) is stored in config field - no separate cascade needed
        };
    }

    protected async deleteRelatedRecords(
        record: any,
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

        // Node links and nodes are now part of the config JSONB field
        // They are automatically deleted when the resource_version record is deleted
        // No separate cleanup needed for config-based workflow data

        // Delete translations
        if (shouldDelete("translations") && record.translations?.length) {
            await tx.resource_version_translation.deleteMany({
                where: { resourceVersionId: record.id },
            });
        }
    }

    /**
     * Create version history for testing
     */
    async createVersionHistory(routineId: bigint, count = 3): Promise<any[]> {
        const versions: any[] = [];
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
    new RoutineVersionDbFactory(prisma);

// Export the class for type usage
export { RoutineVersionDbFactory as RoutineVersionDbFactoryClass };
