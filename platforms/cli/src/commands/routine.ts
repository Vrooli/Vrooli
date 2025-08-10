import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import chalk from "chalk";
import { promises as fs } from "fs";
import * as path from "path";
import { glob } from "glob";
import cliProgress from "cli-progress";
import { UI } from "../utils/constants.js";
import type { 
    Resource,
    ResourceVersionSearchInput,
    ResourceVersionSearchResult,
    ResourceVersionTranslation,
    ResourceSubType,
} from "@vrooli/shared";
import { endpointsResource, ResourceType, type ResourceVersionEdge } from "@vrooli/shared";
import { BaseCommand } from "./BaseCommand.js";

// Interface definitions for routine operations
interface RoutineTranslation {
    language: string;
    name?: string;
    description?: string;
}

interface RoutineStep {
    name?: string;
    type?: string;
    description?: string;
}

interface RoutineValidationData {
    id?: string;
    publicId?: string;
    resourceType?: string;
    versions?: Array<{
        id?: string;
        resourceSubType?: string;
        config?: unknown;
        translations: RoutineTranslation[];
        configFormSchema?: unknown;
        isActive?: boolean;
        isPrivate?: boolean;
        versionNotes?: string;
    }>;
    tags?: unknown[];
    resourceList?: unknown;
    isInternal?: boolean;
    isPrivate?: boolean;
}

interface RoutineDefinitionStep {
    id: string;
    name: string;
    subroutineId: string;
    type?: string;
    description?: string;
}

interface RoutineGraphConfig {
    __type?: string;
    steps?: RoutineStep[];
    schema?: unknown;
}

interface RoutineSearchResult {
    id: string;
    publicId: string;
    name: string;
    type: string;
    description: string;
    score: number;
}


export class RoutineCommands extends BaseCommand {
    constructor(
        program: Command,
        client: ApiClient,
        config: ConfigManager,
    ) {
        super(program, client, config);
    }

    protected registerCommands(): void {
        const routineCmd = this.program
            .command("routine")
            .description("Manage routines");

        routineCmd
            .command("import <file>")
            .description("Import a routine from a JSON file")
            .option("--dry-run", "Validate without importing")
            .option("--validate", "Perform extensive validation")
            .action(async (file, options) => {
                await this.importRoutine(file, options);
            });

        routineCmd
            .command("import-dir <directory>")
            .description("Import all routines from a directory")
            .option("--dry-run", "Validate without importing")
            .option("--fail-fast", "Stop on first error")
            .option("--pattern <pattern>", "File pattern to match", "*.json")
            .action(async (directory, options) => {
                await this.importDirectory(directory, options);
            });

        routineCmd
            .command("export <routineId>")
            .description("Export a routine to a JSON file")
            .option("-o, --output <file>", "Output file path")
            .action(async (routineId, options) => {
                await this.exportRoutine(routineId, options);
            });

        routineCmd
            .command("list")
            .description("List routines")
            .option("-l, --limit <limit>", "Number of routines to show", "10")
            .option("-s, --search <query>", "Search routines")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .option("--mine", "Show only my routines")
            .action(async (options) => {
                await this.listRoutines(options);
            });

        routineCmd
            .command("get <routineId>")
            .description("Get routine details")
            .action(async (routineId) => {
                await this.getRoutine(routineId);
            });

        routineCmd
            .command("validate <file>")
            .description("Validate a routine JSON file")
            .action(async (file) => {
                await this.validateRoutine(file);
            });

        routineCmd
            .command("run <routineId>")
            .description("Execute a routine")
            .option("-i, --input <json>", "Input data as JSON")
            .option("--watch", "Watch execution progress")
            .action(async (routineId, options) => {
                await this.runRoutine(routineId, options);
            });

        routineCmd
            .command("search <query>")
            .description("Search for routines using semantic similarity")
            .option("-l, --limit <limit>", "Number of results to return", "10")
            .option("-t, --type <type>", "Filter by routine type (e.g., RoutineGenerate, RoutineApi)")
            .option("-f, --format <format>", "Output format (table|json|ids)", "table")
            .option("--min-score <score>", "Minimum similarity score (0-1)", "0.7")
            .action(async (query, options) => {
                await this.searchRoutines(query, options);
            });

        routineCmd
            .command("discover")
            .description("List all available routines for use as subroutines")
            .option("--type <type>", "Filter by routine type")
            .option("--format <format>", "Output format (table|json|mapping)", "table")
            .action(async (options) => {
                await this.discoverRoutines(options);
            });
    }

    private async importRoutine(filePath: string, options: { dryRun?: boolean; validate?: boolean }): Promise<void> {
        try {
            // Read and parse file
            const routineData = await this.executeWithSpinner(
                "Reading and parsing routine file...",
                async () => this.parseJsonFile<RoutineValidationData>(filePath),
                "File parsed successfully",
            );

            // Validate structure
            const validation = await this.executeWithSpinner(
                "Validating routine structure...",
                async () => this.validateRoutineData(routineData, options.validate),
                "Validation completed",
            );
            
            if (!validation.valid) {
                output.error("Validation errors:");
                validation.errors.forEach((error: string) => {
                    output.error(`  - ${error}`);
                });
                process.exit(1);
            }

            if (options.dryRun) {
                output.success("Routine is valid");
                const firstVersion = routineData.versions?.[0];
                const name = firstVersion?.translations?.[0]?.name || "Untitled";
                const description = firstVersion?.translations?.[0]?.description || "(none)";
                output.info(`  Name: ${name}`);
                output.info(`  Description: ${description}`);
                output.info(`  Versions: ${routineData.versions?.length || 0}`);
                return;
            }

            // Import to server
            const response = await this.executeWithSpinner(
                "Uploading to server...",
                async () => this.client.requestWithEndpoint<Resource>(
                    endpointsResource.createOne,
                    routineData as Record<string, unknown>,
                ),
                "Routine imported successfully",
            );

            this.output(
                response,
                () => {
                    output.success("Import successful");
                    output.info(`  ID: ${response.id}`);
                    output.info(`  Name: ${response.translatedName || "Untitled"}`);
                    output.info(`  Versions: ${response.versionsCount || 0}`);
                    output.info(`  URL: ${this.config.getServerUrl()}/routine/${response.id}`);
                },
            );
        } catch (error) {
            this.handleError(error, "Import failed");
        }
    }

    private async importDirectory(directory: string, options: {
        dryRun?: boolean;
        failFast?: boolean;
        pattern?: string;
    }): Promise<void> {
        try {
            const absolutePath = path.resolve(directory);
            
            // Find all matching files
            const pattern = path.join(absolutePath, options.pattern || "*.json");
            const files = await glob(pattern);

            if (files.length === 0) {
                output.info(chalk.yellow(`⚠ No files found matching pattern: ${pattern}`));
                return;
            }

            output.info(chalk.bold(`\nFound ${files.length} file(s) to import`));

            // Create progress bar
            const progressBar = new cliProgress.SingleBar({
                format: "Importing |{bar}| {percentage}% | {value}/{total} | {filename}",
                barCompleteChar: "\u2588",
                barIncompleteChar: "\u2591",
                hideCursor: true,
            });

            const results = {
                success: [] as string[],
                failed: [] as { file: string; error: string }[],
            };

            progressBar.start(files.length, 0, { filename: "" });

            for (let i = 0; i < files.length; i++) {
                const file = files[i];
                const filename = path.basename(file);
                progressBar.update(i, { filename });

                try {
                    // Import each file
                    await this.importRoutineSilent(file, options.dryRun);
                    results.success.push(file);
                } catch (error) {
                    const errorMessage = error instanceof Error ? error.message : String(error);
                    results.failed.push({
                        file,
                        error: errorMessage,
                    });

                    if (options.failFast) {
                        progressBar.stop();
                        throw new Error(`Import failed for ${filename}: ${errorMessage}`);
                    }
                }
            }

            progressBar.update(files.length, { filename: "Complete" });
            progressBar.stop();

            // Show results
            this.output(
                results,
                () => {
                    output.info(chalk.bold("\n📊 Import Summary:"));
                    output.success(`  Success: ${results.success.length}`);
                    if (results.failed.length > 0) {
                        output.error(`  Failed: ${results.failed.length}`);
                        output.info(chalk.bold("\n❌ Failed imports:"));
                        results.failed.forEach(({ file, error }) => {
                            output.error(`  ${path.basename(file)}: ${error}`);
                        });
                    }
                },
            );

            // Exit with error if any failed
            if (results.failed.length > 0) {
                throw new Error(`Directory import failed: ${results.failed.length} file(s) failed to import`);
            }
        } catch (error) {
            this.handleError(error, "Directory import failed");
        }
    }

    private async exportRoutine(routineId: string, options: { output?: string }): Promise<void> {
        try {
            const routine = await this.executeWithSpinner(
                "Fetching routine...",
                async () => this.client.requestWithEndpoint<Resource>(
                    endpointsResource.findOne,
                    { publicId: routineId },
                ),
                "Routine fetched",
            );

            // Determine output path
            const outputPath = options.output || `routine-${routineId}.json`;
            const absolutePath = path.resolve(outputPath);

            // Write to file
            await this.writeJsonFile(absolutePath, routine);

            output.success(`Routine exported to: ${absolutePath}`);
        } catch (error) {
            this.handleError(error, "Export failed");
        }
    }

    private async listRoutines(options: {
        limit?: string;
        search?: string;
        format?: string;
        mine?: boolean;
    }): Promise<void> {
        try {
            const params: ResourceVersionSearchInput = {
                take: parseInt(options.limit || "10"),
                isLatest: true,
                // Filter for resources that have "Routine" as their root type
                rootResourceType: ResourceType.Routine,
            };

            if (options.search) {
                params.searchString = options.search;
            }

            if (options.mine) {
                params.createdByIdRoot = "me";
            }

            const response = await this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                endpointsResource.findMany,
                params as Record<string, unknown>,
            );

            if (options.format === "json") {
                output.json(response);
                return;
            }

            this.output(
                response,
                () => {
                    if (!response || !response.edges || response.edges.length === 0) {
                        output.info(chalk.yellow("No routines found"));
                        return;
                    }

                    output.info(chalk.bold("\nRoutines found:\n"));

                    response.edges.forEach((edge, index: number) => {
                        const resource = edge.node;
                        output.info(chalk.cyan(`${index + 1}. ${resource.translations?.[0]?.name || "Untitled"}`));
                        output.info(`   ID: ${resource.id}`);
                        output.info(`   Type: ${resource.resourceSubType}`);
                        output.info(`   Created: ${new Date(resource.createdAt).toLocaleDateString()}`);
                        output.info(`   Version: ${resource.versionLabel}`);
                        output.info("");
                    });

                    if (response.pageInfo.hasNextPage) {
                        output.info(chalk.gray("More results available. Use pagination to see more."));
                    }
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to list routines");
        }
    }

    private async getRoutine(routineId: string): Promise<void> {
        try {
            const routine = await this.executeWithSpinner(
                "Fetching routine...",
                async () => this.client.requestWithEndpoint<Resource>(
                    endpointsResource.findOne,
                    { publicId: routineId },
                ),
                "Routine fetched",
            );

            this.output(
                routine,
                () => {
                    // Get latest version info
                    const latestVersion = routine.versions?.find(v => v.isLatest) || routine.versions?.[0];
                    const versionLabel = latestVersion?.versionLabel || "1.0.0";
                    
                    // Extract name and description from version translations
                    const translation = latestVersion?.translations?.find((t: ResourceVersionTranslation) => t.language === "en") || latestVersion?.translations?.[0];
                    const name = translation?.name || "Untitled";
                    const description = translation?.description || "(none)";

                    output.info(chalk.bold("\nRoutine Details:"));
                    output.info(`  ID: ${routine.id}`);
                    output.info(`  Name: ${name}`);
                    output.info(`  Description: ${description}`);
                    output.info(`  Version: ${versionLabel}`);
                    output.info(`  Created: ${routine.createdAt ? new Date(routine.createdAt).toLocaleString() : "Unknown"}`);
                    output.info(`  Updated: ${routine.updatedAt ? new Date(routine.updatedAt).toLocaleString() : "Unknown"}`);
                    
                    // Note: steps, inputs, and outputs would be in the version's config
                    if (latestVersion?.config) {
                        const config = latestVersion.config as unknown as Record<string, unknown>;
                        
                        // For graph-based routines
                        if (config.graph && typeof config.graph === "object") {
                            const graph = config.graph as Record<string, unknown>;
                            const schema = graph.schema as Record<string, unknown>;
                            
                            if (schema?.steps && Array.isArray(schema.steps)) {
                                output.info(chalk.bold("\nSteps:"));
                                schema.steps.forEach((step: RoutineDefinitionStep, index: number) => {
                                    output.info(`  ${index + 1}. ${step.name || step.type || "Step"}`);
                                    if (step.description) {
                                        output.info(`     ${step.description}`);
                                    }
                                });
                            }
                        }
                        
                        // Note: inputs/outputs would typically be part of the routine's interface
                        // but the Resource type doesn't have these fields directly
                    }
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to fetch routine");
        }
    }

    private async validateRoutine(filePath: string): Promise<void> {
        output.info(chalk.bold("🔍 Validating routine file...\n"));

        try {
            // Parse JSON
            const routineData = await this.parseJsonFile<RoutineValidationData>(filePath);

            // Perform validation
            const validation = await this.validateRoutineData(routineData, true);

            if (validation.valid) {
                output.success("Routine is valid!");
                const firstVersion = routineData.versions?.[0];
                const name = firstVersion?.translations?.[0]?.name || "Untitled";
                output.info(`\n  Name: ${name}`);
                output.info(`  Versions: ${routineData.versions?.length || 0}`);
                output.info(`  Type: ${firstVersion?.resourceSubType || "Unknown"}`);
            } else {
                output.error("Validation failed:\n");
                validation.errors.forEach((error: string) => {
                    output.error(`  - ${error}`);
                });
                // Use handleError to ensure proper error handling in tests
                this.handleError(new Error(`Validation failed with ${validation.errors.length} error(s)`), "Validation error");
            }
        } catch (error) {
            this.handleError(error, "Validation error");
        }
    }

    private async runRoutine(routineId: string, options: { input?: string; watch?: boolean }): Promise<void> {
        try {
            let inputData = {};
            if (options.input) {
                try {
                    inputData = JSON.parse(options.input);
                } catch (error) {
                    output.error("Invalid input JSON");
                    process.exit(1);
                }
            }


            // Start execution
            const execution = await this.executeWithSpinner(
                "Starting routine execution...",
                async () => this.client.post<{ id: string }>("/task/start/run", {
                    routineId,
                    inputs: inputData,
                }),
                "Execution started",
            );

            output.info(`Execution ID: ${execution.id}`);

            if (!options.watch) {
                output.info(chalk.gray("\nRun with --watch to monitor progress"));
                return;
            }

            // Watch execution progress
            output.info(chalk.bold("\n📊 Monitoring execution...\n"));
            
            const socket = this.client.connectWebSocket();
            socket.emit("run:watch", { runId: execution.id });

            const progressBar = new cliProgress.SingleBar({
                format: "Progress |{bar}| {percentage}% | Step: {step}",
                barCompleteChar: "\u2588",
                barIncompleteChar: "\u2591",
            });

            progressBar.start(100, 0, { step: "Initializing..." });

            socket.on(`run:${execution.id}:progress`, (data: { percent: number; currentStep: string }) => {
                progressBar.update(data.percent, { step: data.currentStep });
            });

            socket.on(`run:${execution.id}:log`, (data: { timestamp: string; message: string }) => {
                output.info(`\n[${data.timestamp}] ${data.message}`);
            });

            socket.on(`run:${execution.id}:complete`, (data: { duration: number; status: string; outputs?: unknown }) => {
                progressBar.update(100, { step: "Complete" });
                progressBar.stop();
                
                output.success("Execution complete!");
                output.info(`  Duration: ${data.duration}ms`);
                output.info(`  Status: ${data.status}`);
                
                if (data.outputs) {
                    output.info(chalk.bold("\nOutputs:"));
                    output.json(data.outputs);
                }
                
                socket.disconnect();
                process.exit(0);
            });

            socket.on(`run:${execution.id}:error`, (data: { message: string }) => {
                progressBar.stop();
                output.error(`Execution failed: ${data.message}`);
                socket.disconnect();
                process.exit(1);
            });
        } catch (error) {
            this.handleError(error, "Failed to run routine");
        }
    }

    private async importRoutineSilent(filePath: string, dryRun?: boolean): Promise<{ success: boolean; error?: string; routine?: unknown }> {
        const fileContent = await fs.readFile(filePath, "utf-8");
        const routineData = JSON.parse(fileContent);
        
        const validation = await this.validateRoutineData(routineData, false);
        if (!validation.valid) {
            throw new Error(`Validation failed: ${validation.errors.join(", ")}`);
        }

        if (!dryRun) {
            const result = await this.client.requestWithEndpoint(
                endpointsResource.createOne,
                routineData,
            );
            return { success: true, routine: result };
        }
        
        return { success: true };
    }

    private async validateRoutineData(data: RoutineValidationData, extensive?: boolean): Promise<{
        valid: boolean;
        errors: string[];
    }> {
        const errors: string[] = [];

        // Basic validation - check top-level structure
        if (!data.id || typeof data.id !== "string") {
            errors.push("Missing or invalid 'id' field");
        }

        if (!data.publicId || typeof data.publicId !== "string") {
            errors.push("Missing or invalid 'publicId' field");
        }

        if (!data.resourceType || data.resourceType !== "Routine") {
            errors.push("Missing or invalid 'resourceType' field (must be 'Routine')");
        }

        if (!data.versions || !Array.isArray(data.versions) || data.versions.length === 0) {
            errors.push("Missing or invalid 'versions' array");
            return { valid: false, errors };
        }

        // Validate first version (main validation)
        const version = data.versions[0];
        if (!version.id || typeof version.id !== "string") {
            errors.push("Version missing or invalid 'id' field");
        }

        if (!version.resourceSubType || typeof version.resourceSubType !== "string") {
            errors.push("Version missing or invalid 'resourceSubType' field");
        }

        if (!version.config || typeof version.config !== "object") {
            errors.push("Version missing or invalid 'config' object");
            return { valid: false, errors };
        }

        // Validate config structure based on routine type
        const config = version.config as Record<string, unknown>;
        if (!config.__version) {
            errors.push("Config missing '__version' field");
        }

        // Check for multi-step routines (Sequential or BPMN)
        if (config.graph) {
            await this.validateGraphConfig(config.graph, errors, extensive);
        }
        // Check for single-step routines (Action, API, Generate, etc.)
        else if (config.callDataAction || config.callDataApi || config.callDataGenerate || 
                 config.callDataCode || config.callDataSmartContract || config.callDataWeb) {
            this.validateSingleStepConfig(config, errors);
        } else {
            errors.push("Config missing graph (multi-step) or callData* (single-step) configuration");
        }

        // Validate form schemas
        if (config.formInput && !this.validateFormSchema(config.formInput)) {
            errors.push("Invalid formInput schema");
        }

        if (config.formOutput && !this.validateFormSchema(config.formOutput)) {
            errors.push("Invalid formOutput schema");
        }

        // Validate translations
        if (!version.translations || !Array.isArray(version.translations) || version.translations.length === 0) {
            errors.push("Version missing translations array");
        } else {
            const englishTranslation = version.translations.find((t: RoutineTranslation) => t.language === "en");
            if (!englishTranslation || !englishTranslation.name) {
                errors.push("Missing English translation with name");
            }
        }

        return {
            valid: errors.length === 0,
            errors,
        };
    }

    private async validateGraphConfig(graph: RoutineGraphConfig, errors: string[], extensive?: boolean): Promise<void> {
        if (!graph.__type) {
            errors.push("Graph missing '__type' field");
            return;
        }

        if (!graph.schema) {
            errors.push("Graph missing 'schema' object");
            return;
        }

        const schema = graph.schema as Record<string, unknown>;

        if (graph.__type === "Sequential") {
            if (!schema.steps || !Array.isArray(schema.steps)) {
                errors.push("Sequential graph missing 'steps' array");
            } else {
                // Track all subroutine IDs that need validation
                const subroutineIds: { stepIndex: number; id: string }[] = [];

                schema.steps.forEach((step: RoutineDefinitionStep, index: number) => {
                    if (!step.id || !step.name || !step.subroutineId) {
                        errors.push(`Step ${index + 1}: missing required fields (id, name, subroutineId)`);
                    } else if (step.subroutineId) {
                        // Check if it's a TODO placeholder
                        if (step.subroutineId.startsWith("TODO:")) {
                            errors.push(`Step ${index + 1}: Contains TODO placeholder - subroutine ID needs to be replaced with actual ID`);
                        } else {
                            subroutineIds.push({ stepIndex: index, id: step.subroutineId });
                        }
                    }
                });

                // If extensive validation is enabled and we have a client, check subroutine existence
                if (extensive && subroutineIds.length > 0 && this.client) {
                    output.info(chalk.gray("\n  Checking subroutine existence..."));
                    
                    for (const { stepIndex, id } of subroutineIds) {
                        try {
                            // Try to fetch the subroutine
                            await this.client.requestWithEndpoint(
                                endpointsResource.findOne,
                                { publicId: id },
                            );
                            output.info(chalk.gray(`    ✓ Step ${stepIndex + 1}: Subroutine '${id}' exists`));
                        } catch (error) {
                            errors.push(`Step ${stepIndex + 1}: Subroutine '${id}' not found in database`);
                            output.error(`    Step ${stepIndex + 1}: Subroutine '${id}' not found`);
                        }
                    }
                }
            }

            if (!schema.rootContext) {
                errors.push("Sequential graph missing 'rootContext'");
            }
        } else if (graph.__type === "BPMN-2.0") {
            if (!schema.data || typeof schema.data !== "string") {
                errors.push("BPMN graph missing XML 'data' field");
            }

            if (!schema.activityMap || typeof schema.activityMap !== "object") {
                errors.push("BPMN graph missing 'activityMap'");
            } else if (extensive) {
                // Validate subroutines in activityMap
                const activityIds: { activityId: string; subroutineId: string }[] = [];
                
                for (const [activityId, activity] of Object.entries(schema.activityMap)) {
                    const activityWithSubroutine = activity as { subroutineId?: string };
                    if (activityWithSubroutine.subroutineId) {
                        const subroutineId = activityWithSubroutine.subroutineId;
                        if (subroutineId.startsWith("TODO:")) {
                            errors.push(`Activity '${activityId}': Contains TODO placeholder - subroutine ID needs to be replaced`);
                        } else {
                            activityIds.push({ activityId, subroutineId });
                        }
                    }
                }

                // Check existence if we have activities to validate
                if (activityIds.length > 0 && this.client) {
                    output.info(chalk.gray("\n  Checking BPMN subroutine existence..."));
                    
                    for (const { activityId, subroutineId } of activityIds) {
                        try {
                            await this.client.requestWithEndpoint(
                                endpointsResource.findOne,
                                { publicId: subroutineId },
                            );
                            output.info(chalk.gray(`    ✓ Activity '${activityId}': Subroutine '${subroutineId}' exists`));
                        } catch (error) {
                            errors.push(`Activity '${activityId}': Subroutine '${subroutineId}' not found in database`);
                            output.error(`    Activity '${activityId}': Subroutine '${subroutineId}' not found`);
                        }
                    }
                }
            }
        } else {
            errors.push(`Unknown graph type: ${graph.__type}`);
        }
    }

    private validateSingleStepConfig(config: unknown, errors: string[]): void {
        const cfg = config as Record<string, unknown>;
        const hasCallData = cfg.callDataAction || cfg.callDataApi || cfg.callDataGenerate || 
                           cfg.callDataCode || cfg.callDataSmartContract || cfg.callDataWeb;
        
        if (!hasCallData) {
            errors.push("Single-step routine missing callData configuration");
            return;
        }

        // Validate the specific callData structure
        Object.keys(cfg).forEach(key => {
            if (key.startsWith("callData")) {
                const callData = cfg[key];
                if (typeof callData === "object" && callData !== null) {
                    const callDataObj = callData as Record<string, unknown>;
                    if (!callDataObj.__version || !callDataObj.schema) {
                        errors.push(`${key} missing __version or schema`);
                    }
                }
            }
        });
    }

    private validateFormSchema(form: { __version?: string; schema?: unknown }): boolean {
        const schema = form.schema as Record<string, unknown>;
        return !!(form.__version && schema && 
               Array.isArray(schema.elements) && 
               Array.isArray(schema.containers));
    }

    private async searchRoutines(query: string, options: { limit?: string; type?: string; format?: string; minScore?: string }): Promise<void> {
        try {
            const searchInput = {
                take: parseInt(options.limit || "10"),
                searchString: query,
                where: {
                    ...(options.type && { resourceSubType: { equals: options.type } }),
                    isComplete: { equals: true },
                    isPrivate: { equals: false },
                },
            };

            const response = await this.executeWithSpinner(
                `Searching for routines: "${query}"...`,
                async () => this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                    endpointsResource.findMany,
                    searchInput,
                ),
                "Search complete",
            );

            if (!response.edges || response.edges.length === 0) {
                output.info(`No routines found for "${query}"`);
                return;
            }

            output.info(`Found ${response.edges.length} routine(s) for "${query}"`);

            const routines = response.edges.map((edge: ResourceVersionEdge) => ({
                id: edge.node.id,
                publicId: edge.node.publicId,
                name: edge.node.translations?.[0]?.name || "Unnamed",
                type: edge.node.resourceSubType || "Unknown",
                description: edge.node.translations?.[0]?.description || "No description",
                score: 1.0, // searchScore not available in current API
            })).filter((routine: RoutineSearchResult) => {
                const minScore = parseFloat(options.minScore || "0.7");
                return routine.score >= minScore;
            });

            if (routines.length === 0) {
                output.info(chalk.yellow(`No routines found with similarity score >= ${options.minScore || "0.7"}`));
                return;
            }

            if (options.format === "json") {
                output.json(routines);
            } else if (options.format === "ids") {
                output.info("\n" + chalk.cyan("Routine IDs (for copy/paste):"));
                routines.forEach((routine: { publicId: string }) => {
                    output.info(`"${routine.publicId}"`);
                });
            } else {
                const tableData = routines.map((routine) => ({
                    Score: routine.score.toFixed(2),
                    ID: routine.publicId,
                    Name: routine.name.substring(0, UI.PADDING.BOTTOM - 2),
                    Type: routine.type,
                    Description: routine.description.substring(0, UI.DESCRIPTION_WIDTH),
                }));
                output.table(tableData);
            }
        } catch (error) {
            this.handleError(error, "Failed to search routines");
        }
    }

    private async discoverRoutines(options: { type?: string; format?: string }): Promise<void> {
        try {
            const response = await this.executeWithSpinner(
                "Discovering available routines...",
                async () => this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                    endpointsResource.findMany,
                    {
                        take: 100,
                        isLatest: true,
                        rootResourceType: ResourceType.Routine,
                        ...(options.type && { resourceSubType: options.type as ResourceSubType }),
                        isCompleteWithRoot: true,
                    },
                ),
                "Discovery complete",
            );

            if (!response.edges || response.edges.length === 0) {
                output.info("No routines found");
                return;
            }

            output.info(`Found ${response.edges.length} routine(s)`);

            const routines = response.edges.map((edge: ResourceVersionEdge) => ({
                id: edge.node.id,
                publicId: edge.node.publicId,
                name: edge.node.translations?.[0]?.name || "Unnamed",
                type: edge.node.resourceSubType || "Unknown",
                description: edge.node.translations?.[0]?.description || "No description",
            }));

            if (options.format === "json") {
                output.json(routines);
            } else if (options.format === "mapping") {
                output.info("\n" + chalk.cyan("ID Mapping Reference:"));
                routines.forEach((routine) => {
                    output.info(`"${routine.publicId}" # ${routine.name} (${routine.type})`);
                });
            } else {
                const tableData = routines.map((routine) => ({
                    ID: routine.publicId,
                    Name: routine.name.substring(0, UI.PADDING.BOTTOM - 2),
                    Type: routine.type,
                    Description: routine.description.substring(0, UI.DESCRIPTION_WIDTH),
                }));
                output.table(tableData);
            }
        } catch (error) {
            this.handleError(error, "Failed to discover routines");
        }
    }

}
