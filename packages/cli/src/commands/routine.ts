import { Command } from "commander";
import { ApiClient } from "../utils/client.js";
import { ConfigManager } from "../utils/config.js";
import chalk from "chalk";
import ora from "ora";
import { promises as fs } from "fs";
import * as path from "path";
import { glob } from "glob";
import cliProgress from "cli-progress";

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
                console.error(chalk.red(`✗ JSON parse error: ${error.message}`));
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
            const response = await this.client.post("/resource", routineData);

            spinner.succeed("Routine imported successfully");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(response));
            } else {
                console.log(chalk.green("\n✓ Import successful"));
                console.log(`  ID: ${response.id}`);
                console.log(`  Name: ${response.name}`);
                console.log(`  Version: ${response.version || "1.0.0"}`);
                console.log(`  URL: ${this.config.getServerUrl()}/routine/${response.id}`);
            }
        } catch (error) {
            spinner.fail("Import failed");
            
            if (error.code === "ENOENT") {
                console.error(chalk.red(`✗ File not found: ${filePath}`));
            } else if (error.details) {
                console.error(chalk.red(`✗ Server error: ${error.message}`));
                if (this.config.isDebug() && error.details) {
                    console.error("Details:", error.details);
                }
            } else {
                console.error(chalk.red(`✗ ${error.message}`));
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
                    results.failed.push({
                        file,
                        error: error.message,
                    });

                    if (options.failFast) {
                        progressBar.stop();
                        throw new Error(`Import failed for ${filename}: ${error.message}`);
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
            console.error(chalk.red(`✗ Directory import failed: ${error.message}`));
            process.exit(1);
        }
    }

    private async exportRoutine(routineId: string, options: { output?: string }): Promise<void> {
        const spinner = ora("Fetching routine...").start();

        try {
            const routine = await this.client.get(`/api/routine/${routineId}`);
            spinner.succeed("Routine fetched");

            // Determine output path
            const outputPath = options.output || `routine-${routineId}.json`;
            const absolutePath = path.resolve(outputPath);

            // Write to file
            await fs.writeFile(absolutePath, JSON.stringify(routine, null, 2));

            console.log(chalk.green(`✓ Routine exported to: ${absolutePath}`));
        } catch (error) {
            spinner.fail("Export failed");
            console.error(chalk.red(`✗ ${error.message}`));
            process.exit(1);
        }
    }

    private async listRoutines(options: {
        limit?: string;
        search?: string;
        mine?: boolean;
    }): Promise<void> {
        try {
            const params: any = {
                limit: parseInt(options.limit || "10"),
                offset: 0,
            };

            if (options.search) {
                params.search = options.search;
            }

            if (options.mine) {
                params.userId = "me";
            }

            const response = await this.client.get("/api/routines", { params });

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(response));
                return;
            }

            if (response.items.length === 0) {
                console.log(chalk.yellow("No routines found"));
                return;
            }

            console.log(chalk.bold(`\nRoutines (${response.items.length} of ${response.total}):\n`));

            response.items.forEach((routine: any, index: number) => {
                console.log(chalk.cyan(`${index + 1}. ${routine.name}`));
                console.log(`   ID: ${routine.id}`);
                console.log(`   Description: ${routine.description || "(none)"}`);
                console.log(`   Created: ${new Date(routine.created_at).toLocaleDateString()}`);
                console.log(`   Steps: ${routine.steps?.length || 0}`);
                console.log("");
            });

            if (response.total > response.items.length) {
                console.log(chalk.gray(`Showing ${response.items.length} of ${response.total} routines`));
            }
        } catch (error) {
            console.error(chalk.red(`✗ Failed to list routines: ${error.message}`));
            process.exit(1);
        }
    }

    private async getRoutine(routineId: string): Promise<void> {
        const spinner = ora("Fetching routine...").start();

        try {
            const routine = await this.client.get(`/api/routine/${routineId}`);
            spinner.succeed("Routine fetched");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(routine));
                return;
            }

            console.log(chalk.bold("\nRoutine Details:"));
            console.log(`  ID: ${routine.id}`);
            console.log(`  Name: ${routine.name}`);
            console.log(`  Description: ${routine.description || "(none)"}`);
            console.log(`  Version: ${routine.version || "1.0.0"}`);
            console.log(`  Created: ${new Date(routine.created_at).toLocaleString()}`);
            console.log(`  Updated: ${new Date(routine.updated_at).toLocaleString()}`);
            
            if (routine.steps?.length > 0) {
                console.log(chalk.bold("\nSteps:"));
                routine.steps.forEach((step: any, index: number) => {
                    console.log(`  ${index + 1}. ${step.name || step.type}`);
                    if (step.description) {
                        console.log(`     ${step.description}`);
                    }
                });
            }

            if (routine.inputs?.length > 0) {
                console.log(chalk.bold("\nInputs:"));
                routine.inputs.forEach((input: any) => {
                    console.log(`  - ${input.name} (${input.type})`);
                });
            }

            if (routine.outputs?.length > 0) {
                console.log(chalk.bold("\nOutputs:"));
                routine.outputs.forEach((output: any) => {
                    console.log(`  - ${output.name} (${output.type})`);
                });
            }
        } catch (error) {
            spinner.fail("Failed to fetch routine");
            console.error(chalk.red(`✗ ${error.message}`));
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
                console.error(`  ${error.message}`);
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
            console.error(chalk.red(`✗ Validation error: ${error.message}`));
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
            const execution = await this.client.post("/task/start/run", {
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

            socket.on(`run:${execution.id}:progress`, (data) => {
                progressBar.update(data.percent, { step: data.currentStep });
            });

            socket.on(`run:${execution.id}:log`, (data) => {
                console.log(`\n[${data.timestamp}] ${data.message}`);
            });

            socket.on(`run:${execution.id}:complete`, (data) => {
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

            socket.on(`run:${execution.id}:error`, (data) => {
                progressBar.stop();
                console.error(chalk.red(`\n✗ Execution failed: ${data.error}`));
                socket.disconnect();
                process.exit(1);
            });
        } catch (error) {
            console.error(chalk.red(`✗ Failed to run routine: ${error.message}`));
            process.exit(1);
        }
    }

    private async importRoutineSilent(filePath: string, dryRun?: boolean): Promise<any> {
        const fileContent = await fs.readFile(filePath, "utf-8");
        const routineData = JSON.parse(fileContent);
        
        const validation = await this.validateRoutineData(routineData, false);
        if (!validation.valid) {
            throw new Error(`Validation failed: ${validation.errors.join(", ")}`);
        }

        if (!dryRun) {
            return await this.client.post("/api/routine", routineData);
        }
    }

    private async validateRoutineData(data: any, extensive?: boolean): Promise<{
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
        const config = version.config;
        if (!config.__version) {
            errors.push("Config missing '__version' field");
        }

        // Check for multi-step routines (Sequential or BPMN)
        if (config.graph) {
            this.validateGraphConfig(config.graph, errors);
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
            const englishTranslation = version.translations.find(t => t.language === "en");
            if (!englishTranslation || !englishTranslation.name) {
                errors.push("Missing English translation with name");
            }
        }

        return {
            valid: errors.length === 0,
            errors,
        };
    }

    private validateGraphConfig(graph: any, errors: string[]): void {
        if (!graph.__type) {
            errors.push("Graph missing '__type' field");
            return;
        }

        if (!graph.schema) {
            errors.push("Graph missing 'schema' object");
            return;
        }

        const schema = graph.schema;

        if (graph.__type === "Sequential") {
            if (!schema.steps || !Array.isArray(schema.steps)) {
                errors.push("Sequential graph missing 'steps' array");
            } else {
                schema.steps.forEach((step: any, index: number) => {
                    if (!step.id || !step.name || !step.subroutineId) {
                        errors.push(`Step ${index + 1}: missing required fields (id, name, subroutineId)`);
                    }
                });
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
            }
        } else {
            errors.push(`Unknown graph type: ${graph.__type}`);
        }
    }

    private validateSingleStepConfig(config: any, errors: string[]): void {
        const hasCallData = config.callDataAction || config.callDataApi || config.callDataGenerate || 
                           config.callDataCode || config.callDataSmartContract || config.callDataWeb;
        
        if (!hasCallData) {
            errors.push("Single-step routine missing callData configuration");
            return;
        }

        // Validate the specific callData structure
        Object.keys(config).forEach(key => {
            if (key.startsWith("callData")) {
                const callData = config[key];
                if (!callData.__version || !callData.schema) {
                    errors.push(`${key} missing __version or schema`);
                }
            }
        });
    }

    private validateFormSchema(form: any): boolean {
        return form.__version && form.schema && 
               Array.isArray(form.schema.elements) && 
               Array.isArray(form.schema.containers);
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

            const response = await this.client.request("resource_findMany", {
                input: searchInput,
                fieldName: "resources",
            });

            if (!response.edges || response.edges.length === 0) {
                spinner.info(`No routines found for "${query}"`);
                return;
            }

            spinner.succeed(`Found ${response.edges.length} routine(s) for "${query}"`);

            const routines = response.edges.map((edge: any) => ({
                id: edge.node.id,
                publicId: edge.node.publicId,
                name: edge.node.translations?.[0]?.name || "Unnamed",
                type: edge.node.resourceSubType || "Unknown",
                description: edge.node.translations?.[0]?.description || "No description",
                score: edge.searchScore || 1.0,
            })).filter((routine: any) => {
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
                routines.forEach((routine: any) => {
                    console.log(`"${routine.publicId}"`);
                });
            } else {
                console.log("\n" + chalk.cyan("Search Results:"));
                console.log("Score".padEnd(8) + "ID".padEnd(20) + "Name".padEnd(30) + "Type".padEnd(20) + "Description");
                console.log("─".repeat(98));
                
                routines.forEach((routine: any) => {
                    const scoreStr = routine.score.toFixed(2);
                    console.log(
                        scoreStr.padEnd(8) + 
                        routine.publicId.padEnd(20) + 
                        routine.name.substring(0, 28).padEnd(30) + 
                        routine.type.padEnd(20) + 
                        routine.description.substring(0, 40)
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
            const response = await this.client.request("resource_findMany", {
                input: {
                    take: 100,
                    where: {
                        ...(options.type && { resourceSubType: { equals: options.type } }),
                        isComplete: { equals: true },
                        isPrivate: { equals: false },
                    },
                },
                fieldName: "resources",
            });

            if (!response.edges || response.edges.length === 0) {
                spinner.info("No routines found");
                return;
            }

            spinner.succeed(`Found ${response.edges.length} routine(s)`);

            const routines = response.edges.map((edge: any) => ({
                id: edge.node.id,
                publicId: edge.node.publicId,
                name: edge.node.translations?.[0]?.name || "Unnamed",
                type: edge.node.resourceSubType || "Unknown",
                description: edge.node.translations?.[0]?.description || "No description"
            }));

            if (options.format === "json") {
                console.log(JSON.stringify(routines, null, 2));
            } else if (options.format === "mapping") {
                console.log("\n" + chalk.cyan("ID Mapping Reference:"));
                routines.forEach(routine => {
                    console.log(`"${routine.publicId}" # ${routine.name} (${routine.type})`);
                });
            } else {
                console.log("\n" + chalk.cyan("Available Routines:"));
                console.log("ID".padEnd(20) + "Name".padEnd(30) + "Type".padEnd(20) + "Description");
                console.log("─".repeat(90));
                
                routines.forEach(routine => {
                    console.log(
                        routine.publicId.padEnd(20) + 
                        routine.name.substring(0, 28).padEnd(30) + 
                        routine.type.padEnd(20) + 
                        routine.description.substring(0, 40)
                    );
                });
            }
        } catch (error) {
            spinner.fail(`Failed to discover routines: ${error}`);
            throw error;
        }
    }

}