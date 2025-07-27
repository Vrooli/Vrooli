import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import chalk from "chalk";
import ora from "ora";
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
import { endpointsResource, ResourceType } from "@vrooli/shared";

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

interface RoutineSearchEdge {
    node: {
        id: string;
        publicId: string;
        resourceSubType?: string;
        translations?: RoutineTranslation[];
    };
    searchScore?: number;
}

interface RoutineSearchResult {
    id: string;
    publicId: string;
    name: string;
    type: string;
    description: string;
    score: number;
}


export class RoutineCommands {
    constructor(
        private program: Command,
        private client: ApiClient,
        private config: ConfigManager,
    ) {
        this.registerCommands();
    }

    private registerCommands(): void {
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
        const spinner = ora("Reading routine file...").start();

        try {
            // Read and parse file
            const absolutePath = path.resolve(filePath);
            const fileContent = await fs.readFile(absolutePath, "utf-8");
            
            spinner.text = "Parsing JSON...";
            let routineData;
            try {
                routineData = JSON.parse(fileContent);
            } catch (error) {
                spinner.fail("Invalid JSON");
                const errorMessage = error instanceof Error ? error.message : String(error);
                console.error(chalk.red(`✗ JSON parse error: ${errorMessage}`));
                process.exit(1);
            }

            // Validate structure
            spinner.text = "Validating routine structure...";
            const validation = await this.validateRoutineData(routineData, options.validate);
            
            if (!validation.valid) {
                spinner.fail("Validation failed");
                console.error(chalk.red("\n✗ Validation errors:"));
                validation.errors.forEach((error: string) => {
                    console.error(`  - ${error}`);
                });
                process.exit(1);
            }

            if (options.dryRun) {
                spinner.succeed("Validation passed (dry run)");
                console.log(chalk.green("\n✓ Routine is valid"));
                console.log(`  Name: ${routineData.name}`);
                console.log(`  Description: ${routineData.description || "(none)"}`);
                console.log(`  Steps: ${routineData.steps?.length || 0}`);
                return;
            }

            // Import to server
            spinner.text = "Uploading to server...";
            const response = await this.client.requestWithEndpoint<Resource>(
                endpointsResource.createOne,
                routineData,
            );

            spinner.succeed("Routine imported successfully");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(response));
            } else {
                console.log(chalk.green("\n✓ Import successful"));
                console.log(`  ID: ${response.id}`);
                console.log(`  Name: ${response.translatedName || "Untitled"}`);
                console.log(`  Versions: ${response.versionsCount || 0}`);
                console.log(`  URL: ${this.config.getServerUrl()}/routine/${response.id}`);
            }
        } catch (error) {
            spinner.fail("Import failed");
            
            const err = error as { code?: string; message?: string; details?: unknown };
            if (err.code === "ENOENT") {
                console.error(chalk.red(`✗ File not found: ${filePath}`));
            } else if (err.details) {
                const errorMessage = err.message || String(error);
                console.error(chalk.red(`✗ Server error: ${errorMessage}`));
                if (this.config.isDebug() && err.details) {
                    console.error("Details:", err.details);
                }
            } else {
                const errorMessage = error instanceof Error ? error.message : String(error);
                console.error(chalk.red(`✗ ${errorMessage}`));
            }
            process.exit(1);
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
                console.log(chalk.yellow(`⚠ No files found matching pattern: ${pattern}`));
                return;
            }

            console.log(chalk.bold(`\nFound ${files.length} file(s) to import`));

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
            console.log(chalk.bold("\n📊 Import Summary:"));
            console.log(chalk.green(`  ✓ Success: ${results.success.length}`));
            if (results.failed.length > 0) {
                console.log(chalk.red(`  ✗ Failed: ${results.failed.length}`));
                
                if (!this.config.isJsonOutput()) {
                    console.log(chalk.bold("\n❌ Failed imports:"));
                    results.failed.forEach(({ file, error }) => {
                        console.log(`  ${path.basename(file)}: ${error}`);
                    });
                }
            }

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(results));
            }

            // Exit with error if any failed
            if (results.failed.length > 0) {
                process.exit(1);
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Directory import failed: ${errorMessage}`));
            process.exit(1);
        }
    }

    private async exportRoutine(routineId: string, options: { output?: string }): Promise<void> {
        const spinner = ora("Fetching routine...").start();

        try {
            const routine = await this.client.requestWithEndpoint<Resource>(
                endpointsResource.findOne,
                { publicId: routineId },
            );
            spinner.succeed("Routine fetched");

            // Determine output path
            const outputPath = options.output || `routine-${routineId}.json`;
            const absolutePath = path.resolve(outputPath);

            // Write to file
            await fs.writeFile(absolutePath, JSON.stringify(routine, null, 2));

            console.log(chalk.green(`✓ Routine exported to: ${absolutePath}`));
        } catch (error) {
            spinner.fail("Export failed");
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ ${errorMessage}`));
            process.exit(1);
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
                // TODO: ResourceSearchInput doesn't have a direct search field
                // Need to check API for text search capability
            }

            if (options.mine) {
                params.createdById = "me";
            }

            const response = await this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                endpointsResource.findMany,
                params,
            );

            if (this.config.isJsonOutput() || options.format === "json") {
                console.log(JSON.stringify(response));
                return;
            }

            if (!response || !response.edges || response.edges.length === 0) {
                console.log(chalk.yellow("No routines found"));
                return;
            }

            console.log(chalk.bold("\nRoutines found:\n"));

            response.edges.forEach((edge, index: number) => {
                const resource = edge.node;
                console.log(chalk.cyan(`${index + 1}. ${resource.translatedName || "Untitled"}`));
                console.log(`   ID: ${resource.id}`);
                console.log(`   Type: ${resource.resourceType}`);
                console.log(`   Created: ${new Date(resource.createdAt).toLocaleDateString()}`);
                console.log(`   Versions: ${resource.versionsCount || 0}`);
                console.log("");
            });

            if (response.pageInfo.hasNextPage) {
                console.log(chalk.gray("More results available. Use pagination to see more."));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Failed to list routines: ${errorMessage}`));
            process.exit(1);
        }
    }

    private async getRoutine(routineId: string): Promise<void> {
        const spinner = ora("Fetching routine...").start();

        try {
            const routine = await this.client.requestWithEndpoint<Resource>(
                endpointsResource.findOne,
                { publicId: routineId },
            );
            spinner.succeed("Routine fetched");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(routine));
                return;
            }

            // Get latest version info
            const latestVersion = routine.versions?.find(v => v.isLatest) || routine.versions?.[0];
            const versionLabel = latestVersion?.versionLabel || "1.0.0";
            
            // Extract name and description from version translations
            const translation = latestVersion?.translations?.find((t: ResourceVersionTranslation) => t.language === "en") || latestVersion?.translations?.[0];
            const name = translation?.name || "Untitled";
            const description = translation?.description || "(none)";

            console.log(chalk.bold("\nRoutine Details:"));
            console.log(`  ID: ${routine.id}`);
            console.log(`  Name: ${name}`);
            console.log(`  Description: ${description}`);
            console.log(`  Version: ${versionLabel}`);
            console.log(`  Created: ${routine.createdAt ? new Date(routine.createdAt).toLocaleString() : "Unknown"}`);
            console.log(`  Updated: ${routine.updatedAt ? new Date(routine.updatedAt).toLocaleString() : "Unknown"}`);
            
            // Note: steps, inputs, and outputs would be in the version's config
            if (latestVersion?.config) {
                const config = latestVersion.config as unknown as Record<string, unknown>;
                
                // For graph-based routines
                if (config.graph && typeof config.graph === "object") {
                    const graph = config.graph as Record<string, unknown>;
                    const schema = graph.schema as Record<string, unknown>;
                    
                    if (schema?.steps && Array.isArray(schema.steps)) {
                        console.log(chalk.bold("\nSteps:"));
                        schema.steps.forEach((step: RoutineDefinitionStep, index: number) => {
                            console.log(`  ${index + 1}. ${step.name || step.type || "Step"}`);
                            if (step.description) {
                                console.log(`     ${step.description}`);
                            }
                        });
                    }
                }
                
                // Note: inputs/outputs would typically be part of the routine's interface
                // but the Resource type doesn't have these fields directly
            }
        } catch (error) {
            spinner.fail("Failed to fetch routine");
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ ${errorMessage}`));
            process.exit(1);
        }
    }

    private async validateRoutine(filePath: string): Promise<void> {
        console.log(chalk.bold("🔍 Validating routine file...\n"));

        try {
            const absolutePath = path.resolve(filePath);
            const fileContent = await fs.readFile(absolutePath, "utf-8");
            
            // Parse JSON
            let routineData;
            try {
                routineData = JSON.parse(fileContent);
            } catch (error) {
                console.error(chalk.red("✗ Invalid JSON:"));
                const errorMessage = error instanceof Error ? error.message : String(error);
                console.error(`  ${errorMessage}`);
                process.exit(1);
            }

            // Perform validation
            const validation = await this.validateRoutineData(routineData, true);

            if (validation.valid) {
                console.log(chalk.green("✓ Routine is valid!"));
                console.log(`\n  Name: ${routineData.name}`);
                console.log(`  Steps: ${routineData.steps?.length || 0}`);
                console.log(`  Inputs: ${routineData.inputs?.length || 0}`);
                console.log(`  Outputs: ${routineData.outputs?.length || 0}`);
            } else {
                console.error(chalk.red("✗ Validation failed:\n"));
                validation.errors.forEach((error: string) => {
                    console.error(`  - ${error}`);
                });
                process.exit(1);
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Validation error: ${errorMessage}`));
            process.exit(1);
        }
    }

    private async runRoutine(routineId: string, options: { input?: string; watch?: boolean }): Promise<void> {
        try {
            let inputData = {};
            if (options.input) {
                try {
                    inputData = JSON.parse(options.input);
                } catch (error) {
                    console.error(chalk.red("✗ Invalid input JSON"));
                    process.exit(1);
                }
            }

            const spinner = ora("Starting routine execution...").start();

            // Start execution
            const execution = await this.client.post<{ id: string }>("/task/start/run", {
                routineId,
                inputs: inputData,
            });

            spinner.succeed(`Execution started: ${execution.id}`);

            if (!options.watch) {
                console.log(chalk.gray("\nRun with --watch to monitor progress"));
                return;
            }

            // Watch execution progress
            console.log(chalk.bold("\n📊 Monitoring execution...\n"));
            
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
                console.log(`\n[${data.timestamp}] ${data.message}`);
            });

            socket.on(`run:${execution.id}:complete`, (data: { duration: number; status: string; outputs?: unknown }) => {
                progressBar.update(100, { step: "Complete" });
                progressBar.stop();
                
                console.log(chalk.green("\n✓ Execution complete!"));
                console.log(`  Duration: ${data.duration}ms`);
                console.log(`  Status: ${data.status}`);
                
                if (data.outputs) {
                    console.log(chalk.bold("\nOutputs:"));
                    console.log(JSON.stringify(data.outputs, null, 2));
                }
                
                socket.disconnect();
                process.exit(0);
            });

            socket.on(`run:${execution.id}:error`, (data: { message: string }) => {
                progressBar.stop();
                console.error(chalk.red(`\n✗ Execution failed: ${data.message}`));
                socket.disconnect();
                process.exit(1);
            });
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Failed to run routine: ${errorMessage}`));
            process.exit(1);
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
                    console.log(chalk.gray("\n  Checking subroutine existence..."));
                    
                    for (const { stepIndex, id } of subroutineIds) {
                        try {
                            // Try to fetch the subroutine
                            await this.client.requestWithEndpoint(
                                endpointsResource.findOne,
                                { publicId: id },
                            );
                            console.log(chalk.gray(`    ✓ Step ${stepIndex + 1}: Subroutine '${id}' exists`));
                        } catch (error) {
                            errors.push(`Step ${stepIndex + 1}: Subroutine '${id}' not found in database`);
                            console.log(chalk.red(`    ✗ Step ${stepIndex + 1}: Subroutine '${id}' not found`));
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
                    console.log(chalk.gray("\n  Checking BPMN subroutine existence..."));
                    
                    for (const { activityId, subroutineId } of activityIds) {
                        try {
                            await this.client.requestWithEndpoint(
                                endpointsResource.findOne,
                                { publicId: subroutineId },
                            );
                            console.log(chalk.gray(`    ✓ Activity '${activityId}': Subroutine '${subroutineId}' exists`));
                        } catch (error) {
                            errors.push(`Activity '${activityId}': Subroutine '${subroutineId}' not found in database`);
                            console.log(chalk.red(`    ✗ Activity '${activityId}': Subroutine '${subroutineId}' not found`));
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
        const spinner = ora(`Searching for routines: "${query}"...`).start();

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

            const response = await this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                endpointsResource.findMany,
                searchInput,
            );

            if (!response.edges || response.edges.length === 0) {
                spinner.info(`No routines found for "${query}"`);
                return;
            }

            spinner.succeed(`Found ${response.edges.length} routine(s) for "${query}"`);

            const routines = response.edges.map((edge: RoutineSearchEdge) => ({
                id: edge.node.id,
                publicId: edge.node.publicId,
                name: edge.node.translations?.[0]?.name || "Unnamed",
                type: edge.node.resourceSubType || "Unknown",
                description: edge.node.translations?.[0]?.description || "No description",
                score: edge.searchScore || 1.0,
            })).filter((routine: RoutineSearchResult) => {
                const minScore = parseFloat(options.minScore || "0.7");
                return routine.score >= minScore;
            });

            if (routines.length === 0) {
                console.log(chalk.yellow(`No routines found with similarity score >= ${options.minScore || "0.7"}`));
                return;
            }

            if (options.format === "json") {
                console.log(JSON.stringify(routines, null, 2));
            } else if (options.format === "ids") {
                console.log("\n" + chalk.cyan("Routine IDs (for copy/paste):"));
                routines.forEach((routine: { publicId: string }) => {
                    console.log(`"${routine.publicId}"`);
                });
            } else {
                console.log("\n" + chalk.cyan("Search Results:"));
                console.log("Score".padEnd(UI.PADDING.TOP) + "ID".padEnd(UI.PADDING.RIGHT) + "Name".padEnd(UI.PADDING.BOTTOM) + "Type".padEnd(UI.PADDING.LEFT) + "Description");
                console.log("─".repeat(UI.CONTENT_WIDTH));
                
                routines.forEach((routine) => {
                    const scoreStr = routine.score.toFixed(2);
                    console.log(
                        scoreStr.padEnd(UI.PADDING.TOP) + 
                        routine.publicId.padEnd(UI.PADDING.RIGHT) + 
                        routine.name.substring(0, UI.PADDING.BOTTOM - 2).padEnd(UI.PADDING.BOTTOM) + 
                        routine.type.padEnd(UI.PADDING.LEFT) + 
                        routine.description.substring(0, UI.DESCRIPTION_WIDTH),
                    );
                });
            }
        } catch (error) {
            spinner.fail(`Failed to search routines: ${error}`);
            throw error;
        }
    }

    private async discoverRoutines(options: { type?: string; format?: string }): Promise<void> {
        const spinner = ora("Discovering available routines...").start();

        try {
            const response = await this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                endpointsResource.findMany,
                {
                    take: 100,
                    isLatest: true,
                    rootResourceType: ResourceType.Routine,
                    ...(options.type && { resourceSubType: options.type as ResourceSubType }),
                    isCompleteWithRoot: true,
                },
            );

            if (!response.edges || response.edges.length === 0) {
                spinner.info("No routines found");
                return;
            }

            spinner.succeed(`Found ${response.edges.length} routine(s)`);

            const routines = response.edges.map((edge: RoutineSearchEdge) => ({
                id: edge.node.id,
                publicId: edge.node.publicId,
                name: edge.node.translations?.[0]?.name || "Unnamed",
                type: edge.node.resourceSubType || "Unknown",
                description: edge.node.translations?.[0]?.description || "No description",
            }));

            if (options.format === "json") {
                console.log(JSON.stringify(routines, null, 2));
            } else if (options.format === "mapping") {
                console.log("\n" + chalk.cyan("ID Mapping Reference:"));
                routines.forEach((routine) => {
                    console.log(`"${routine.publicId}" # ${routine.name} (${routine.type})`);
                });
            } else {
                console.log("\n" + chalk.cyan("Available Routines:"));
                console.log("ID".padEnd(UI.PADDING.RIGHT) + "Name".padEnd(UI.PADDING.BOTTOM) + "Type".padEnd(UI.PADDING.LEFT) + "Description");
                console.log("─".repeat(UI.CONTENT_WIDTH - UI.WIDTH_ADJUSTMENT));
                
                routines.forEach((routine) => {
                    console.log(
                        routine.publicId.padEnd(UI.PADDING.RIGHT) + 
                        routine.name.substring(0, UI.PADDING.BOTTOM - 2).padEnd(UI.PADDING.BOTTOM) + 
                        routine.type.padEnd(UI.PADDING.LEFT) + 
                        routine.description.substring(0, UI.DESCRIPTION_WIDTH),
                    );
                });
            }
        } catch (error) {
            spinner.fail(`Failed to discover routines: ${error}`);
            throw error;
        }
    }

}
