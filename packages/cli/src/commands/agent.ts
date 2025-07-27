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
    BotConfigObject,
    ResourceCreateInput,
} from "@vrooli/shared";
import { generatePK, ResourceType, ResourceSubType, ResourceUsedFor, endpointsResource } from "@vrooli/shared";

// Interface definitions for agent operations
interface AgentTranslation {
    language: string;
    name?: string;
    description?: string;
}

interface AgentValidationData {
    identity: {
        name: string;
        id?: string;
        version?: string;
    };
    goal: string;
    role: string;
    subscriptions: string[];
    behaviors: Array<{
        trigger: {
            topic: string;
            when?: string;
        };
        action: {
            type: "routine" | "invoke";
            label?: string;
            inputMap?: Record<string, unknown>;
            purpose?: string;
        };
        qos?: number;
    }>;
    prompt?: {
        mode: "supplement" | "replace";
        source: "direct" | "resource";
        content?: string;
        resourceId?: string;
    };
    resources?: string[];
}

interface AgentSearchEdge {
    node: {
        id: string;
        publicId: string;
        resourceSubType?: string;
        translations?: AgentTranslation[];
    };
    searchScore?: number;
}

export class AgentCommands {
    constructor(
        private program: Command,
        private client: ApiClient,
        private config: ConfigManager,
    ) {
        this.registerCommands();
    }

    private registerCommands(): void {
        const agentCmd = this.program
            .command("agent")
            .description("Manage AI agents for swarm orchestration");

        agentCmd
            .command("import <file>")
            .description("Import an agent from a JSON file")
            .option("--dry-run", "Validate without importing")
            .option("--validate", "Perform extensive validation including routine checks")
            .action(async (file, options) => {
                await this.importAgent(file, options);
            });

        agentCmd
            .command("import-dir <directory>")
            .description("Import all agents from a directory")
            .option("--dry-run", "Validate without importing")
            .option("--fail-fast", "Stop on first error")
            .option("--pattern <pattern>", "File pattern to match", "*.json")
            .action(async (directory, options) => {
                await this.importDirectory(directory, options);
            });

        agentCmd
            .command("export <agentId>")
            .description("Export an agent to a JSON file")
            .option("-o, --output <file>", "Output file path")
            .action(async (agentId, options) => {
                await this.exportAgent(agentId, options);
            });

        agentCmd
            .command("list")
            .description("List agents")
            .option("-l, --limit <limit>", "Number of agents to show", "20")
            .option("-s, --search <query>", "Search agents by name or goal")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .option("--mine", "Show only my agents")
            .option("--role <role>", "Filter by role (coordinator|specialist|monitor|bridge)")
            .action(async (options) => {
                await this.listAgents(options);
            });

        agentCmd
            .command("get <agentId>")
            .description("Get agent details")
            .action(async (agentId) => {
                await this.getAgent(agentId);
            });

        agentCmd
            .command("validate <file>")
            .description("Validate an agent JSON file")
            .option("--check-routines", "Validate referenced routine existence")
            .action(async (file, options) => {
                await this.validateAgent(file, options);
            });

        agentCmd
            .command("search <query>")
            .description("Search for agents by goal or capability")
            .option("-l, --limit <limit>", "Number of results to return", "10")
            .option("-r, --role <role>", "Filter by role")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .action(async (query, options) => {
                await this.searchAgents(query, options);
            });
    }

    private async importAgent(filePath: string, options: { dryRun?: boolean; validate?: boolean }): Promise<void> {
        const spinner = ora("Reading agent file...").start();

        try {
            // Read and parse file
            const absolutePath = path.resolve(filePath);
            const fileContent = await fs.readFile(absolutePath, "utf-8");
            
            spinner.text = "Parsing JSON...";
            let agentData;
            try {
                agentData = JSON.parse(fileContent);
            } catch (error) {
                spinner.fail("Invalid JSON");
                const errorMessage = error instanceof Error ? error.message : String(error);
                console.error(chalk.red(`✗ JSON parse error: ${errorMessage}`));
                process.exit(1);
            }

            // Validate structure
            spinner.text = "Validating agent structure...";
            const validation = await this.validateAgentData(agentData, options.validate);
            
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
                console.log(chalk.green("\n✓ Agent is valid"));
                console.log(`  Name: ${agentData.identity.name}`);
                console.log(`  Goal: ${agentData.goal}`);
                console.log(`  Role: ${agentData.role}`);
                console.log(`  Subscriptions: ${agentData.subscriptions.length}`);
                console.log(`  Behaviors: ${agentData.behaviors.length}`);
                return;
            }

            // Convert agent format to Resource format for API
            spinner.text = "Uploading to server...";
            const resourceData = this.convertAgentToResource(agentData);
            const response = await this.client.post<Resource>("/api/resource", resourceData);

            spinner.succeed("Agent imported successfully");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(response));
            } else {
                console.log(chalk.green("\n✓ Import successful"));
                console.log(`  ID: ${response.id}`);
                console.log(`  Name: ${agentData.identity.name}`);
                console.log(`  Role: ${agentData.role}`);
                console.log(`  URL: ${this.config.getServerUrl()}/agent/${response.id}`);
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
            const pattern = path.join(absolutePath, "**", options.pattern || "*.json");
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
                    await this.importAgentSilent(file, options.dryRun);
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

    private async exportAgent(agentId: string, options: { output?: string }): Promise<void> {
        const spinner = ora("Fetching agent...").start();

        try {
            const agent = await this.client.get<Resource>(`/api/resource/${agentId}`);
            spinner.succeed("Agent fetched");

            // Convert back to agent format
            const agentData = this.convertResourceToAgent(agent);

            // Determine output path
            const outputPath = options.output || `agent-${agentData.identity.name}.json`;
            const absolutePath = path.resolve(outputPath);

            // Write to file
            await fs.writeFile(absolutePath, JSON.stringify(agentData, null, 2));

            console.log(chalk.green(`✓ Agent exported to: ${absolutePath}`));
        } catch (error) {
            spinner.fail("Export failed");
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ ${errorMessage}`));
            process.exit(1);
        }
    }

    private async listAgents(options: {
        limit?: string;
        search?: string;
        format?: string;
        mine?: boolean;
        role?: string;
    }): Promise<void> {
        try {
            const params: ResourceVersionSearchInput = {
                take: parseInt(options.limit || "20"),
                latestVersionResourceSubType: "AgentSpec" as ResourceSubType,
            };

            if (options.search) {
                params.searchString = options.search;
            }

            if (options.mine) {
                params.createdById = "me";
            }

            // Note: Role filtering would need to be done post-fetch since it's in metadata
            const response = await this.client.get<ResourceVersionSearchResult>("/api/resources", { params });

            if (this.config.isJsonOutput() || options.format === "json") {
                console.log(JSON.stringify(response));
                return;
            }

            if (!response || !response.edges || response.edges.length === 0) {
                console.log(chalk.yellow("No agents found"));
                return;
            }

            console.log(chalk.bold("\nAgents found:\n"));

            response.edges.forEach((edge, index: number) => {
                const resource = edge.node;
                const metadata = resource.versions?.[0]?.config as BotConfigObject;
                const role = metadata?.agentSpec?.role || "unknown";
                
                // Filter by role if specified
                if (options.role && role !== options.role) {
                    return;
                }

                console.log(chalk.cyan(`${index + 1}. ${resource.translatedName || "Unnamed Agent"}`));
                console.log(`   ID: ${resource.id}`);
                console.log(`   Role: ${role}`);
                console.log(`   Goal: ${metadata?.agentSpec?.goal || "No goal specified"}`);
                console.log(`   Created: ${new Date(resource.createdAt).toLocaleDateString()}`);
                console.log("");
            });

            if (response.pageInfo.hasNextPage) {
                console.log(chalk.gray("More results available. Use pagination to see more."));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Failed to list agents: ${errorMessage}`));
            process.exit(1);
        }
    }

    private async getAgent(agentId: string): Promise<void> {
        const spinner = ora("Fetching agent...").start();

        try {
            const agent = await this.client.get<Resource>(`/api/resource/${agentId}`);
            spinner.succeed("Agent fetched");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(agent));
                return;
            }

            // Extract agent data from resource
            const agentData = this.convertResourceToAgent(agent);

            console.log(chalk.bold("\nAgent Details:"));
            console.log(`  ID: ${agent.id}`);
            console.log(`  Name: ${agentData.identity.name}`);
            console.log(`  Goal: ${agentData.goal}`);
            console.log(`  Role: ${agentData.role}`);
            console.log(`  Created: ${agent.createdAt ? new Date(agent.createdAt).toLocaleString() : "Unknown"}`);
            console.log(`  Updated: ${agent.updatedAt ? new Date(agent.updatedAt).toLocaleString() : "Unknown"}`);
            
            console.log(chalk.bold("\nSubscriptions:"));
            agentData.subscriptions.forEach((sub: string, index: number) => {
                console.log(`  ${index + 1}. ${sub}`);
            });

            console.log(chalk.bold("\nBehaviors:"));
            agentData.behaviors.forEach((behavior, index: number) => {
                console.log(`  ${index + 1}. Trigger: ${behavior.trigger.topic}`);
                if (behavior.trigger.when) {
                    console.log(`     When: ${behavior.trigger.when}`);
                }
                console.log(`     Action: ${behavior.action.type}`);
                if (behavior.action.type === "routine") {
                    console.log(`     Routine: ${behavior.action.label}`);
                } else {
                    console.log(`     Purpose: ${behavior.action.purpose}`);
                }
            });

            if (agentData.prompt) {
                console.log(chalk.bold("\nPrompt:"));
                console.log(`  Mode: ${agentData.prompt.mode}`);
                console.log(`  Source: ${agentData.prompt.source}`);
                if (agentData.prompt.content) {
                    console.log(`  Content: ${agentData.prompt.content.substring(0, 100)}...`);
                }
            }
        } catch (error) {
            spinner.fail("Failed to fetch agent");
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ ${errorMessage}`));
            process.exit(1);
        }
    }

    private async validateAgent(filePath: string, options: { checkRoutines?: boolean }): Promise<void> {
        console.log(chalk.bold("🔍 Validating agent file...\n"));

        try {
            const absolutePath = path.resolve(filePath);
            const fileContent = await fs.readFile(absolutePath, "utf-8");
            
            // Parse JSON
            let agentData;
            try {
                agentData = JSON.parse(fileContent);
            } catch (error) {
                console.error(chalk.red("✗ Invalid JSON:"));
                const errorMessage = error instanceof Error ? error.message : String(error);
                console.error(`  ${errorMessage}`);
                process.exit(1);
            }

            // Perform validation
            const validation = await this.validateAgentData(agentData, options.checkRoutines);

            if (validation.valid) {
                console.log(chalk.green("✓ Agent is valid!"));
                console.log(`\n  Name: ${agentData.identity.name}`);
                console.log(`  Goal: ${agentData.goal}`);
                console.log(`  Role: ${agentData.role}`);
                console.log(`  Subscriptions: ${agentData.subscriptions.length}`);
                console.log(`  Behaviors: ${agentData.behaviors.length}`);
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

    private async searchAgents(query: string, options: {
        limit?: string;
        role?: string;
        format?: string;
    }): Promise<void> {
        const spinner = ora(`Searching for agents: "${query}"...`).start();

        try {
            const searchInput: ResourceVersionSearchInput = {
                take: parseInt(options.limit || "10"),
                searchString: query,
                isLatest: true,
                rootResourceType: ResourceType.Agent,
            };

            const response = await this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                endpointsResource.findMany,
                searchInput,
            );

            if (!response.edges || response.edges.length === 0) {
                spinner.info(`No agents found for "${query}"`);
                return;
            }

            spinner.succeed(`Found ${response.edges.length} agent(s) for "${query}"`);

            const agents = response.edges.map((edge: AgentSearchEdge) => {
                const translation = edge.node.translations?.[0];
                return {
                    id: edge.node.id,
                    publicId: edge.node.publicId,
                    name: translation?.name || "Unnamed",
                    description: translation?.description || "No description",
                    score: edge.searchScore || 1.0,
                };
            });

            if (options.format === "json") {
                console.log(JSON.stringify(agents, null, 2));
            } else {
                console.log("\n" + chalk.cyan("Search Results:"));
                console.log("Score".padEnd(UI.PADDING.TOP) + "ID".padEnd(UI.PADDING.RIGHT) + "Name".padEnd(UI.PADDING.BOTTOM) + "Description");
                console.log("─".repeat(UI.CONTENT_WIDTH));
                
                agents.forEach((agent) => {
                    const scoreStr = agent.score.toFixed(2);
                    console.log(
                        scoreStr.padEnd(UI.PADDING.TOP) + 
                        agent.publicId.padEnd(UI.PADDING.RIGHT) + 
                        agent.name.substring(0, UI.PADDING.BOTTOM - 2).padEnd(UI.PADDING.BOTTOM) + 
                        agent.description.substring(0, UI.DESCRIPTION_WIDTH),
                    );
                });
            }
        } catch (error) {
            spinner.fail(`Failed to search agents: ${error}`);
            throw error;
        }
    }

    private async importAgentSilent(filePath: string, dryRun?: boolean): Promise<{ success: boolean; error?: string; agent?: unknown }> {
        const fileContent = await fs.readFile(filePath, "utf-8");
        const agentData = JSON.parse(fileContent);
        
        const validation = await this.validateAgentData(agentData, false);
        if (!validation.valid) {
            throw new Error(`Validation failed: ${validation.errors.join(", ")}`);
        }

        if (!dryRun) {
            const resourceData = this.convertAgentToResource(agentData);
            const result = await this.client.post("/api/resource", resourceData);
            return { success: true, agent: result };
        }
        
        return { success: true };
    }

    private async validateAgentData(data: AgentValidationData, checkRoutines?: boolean): Promise<{
        valid: boolean;
        errors: string[];
    }> {
        const errors: string[] = [];

        // Validate identity
        if (!data.identity || typeof data.identity !== "object") {
            errors.push("Missing or invalid 'identity' object");
        } else {
            if (!data.identity.name || typeof data.identity.name !== "string") {
                errors.push("Missing or invalid 'identity.name'");
            }
        }

        // Validate required fields
        if (!data.goal || typeof data.goal !== "string") {
            errors.push("Missing or invalid 'goal'");
        }

        if (!data.role || typeof data.role !== "string") {
            errors.push("Missing or invalid 'role'");
        } else if (!["coordinator", "specialist", "monitor", "bridge"].includes(data.role)) {
            errors.push(`Invalid role '${data.role}'. Must be: coordinator, specialist, monitor, or bridge`);
        }

        // Validate subscriptions
        if (!data.subscriptions || !Array.isArray(data.subscriptions)) {
            errors.push("Missing or invalid 'subscriptions' array");
        } else if (data.subscriptions.length === 0) {
            errors.push("Agent must have at least one subscription");
        } else {
            // Validate subscription format
            const validPrefixes = ["swarm/", "run/", "step/", "safety/"];
            data.subscriptions.forEach((sub, index) => {
                if (typeof sub !== "string") {
                    errors.push(`Subscription ${index + 1} is not a string`);
                } else if (!validPrefixes.some(prefix => sub.startsWith(prefix))) {
                    errors.push(`Subscription '${sub}' must start with: swarm/, run/, step/, or safety/`);
                }
            });
        }

        // Validate behaviors
        if (!data.behaviors || !Array.isArray(data.behaviors)) {
            errors.push("Missing or invalid 'behaviors' array");
        } else if (data.behaviors.length === 0) {
            errors.push("Agent must have at least one behavior");
        } else {
            data.behaviors.forEach((behavior, index) => {
                // Validate trigger
                if (!behavior.trigger || !behavior.trigger.topic) {
                    errors.push(`Behavior ${index + 1}: Missing trigger.topic`);
                } else if (!data.subscriptions.includes(behavior.trigger.topic)) {
                    errors.push(`Behavior ${index + 1}: Topic '${behavior.trigger.topic}' not in subscriptions`);
                }

                // Validate action
                if (!behavior.action) {
                    errors.push(`Behavior ${index + 1}: Missing action`);
                } else {
                    if (!behavior.action.type || !["routine", "invoke"].includes(behavior.action.type)) {
                        errors.push(`Behavior ${index + 1}: Invalid action type. Must be 'routine' or 'invoke'`);
                    } else if (behavior.action.type === "routine") {
                        if (!behavior.action.label) {
                            errors.push(`Behavior ${index + 1}: Routine action missing 'label'`);
                        } else if (checkRoutines) {
                            // Would check routine existence here if enabled
                            console.log(chalk.gray(`  Checking routine '${behavior.action.label}'...`));
                        }
                    } else if (behavior.action.type === "invoke") {
                        if (!behavior.action.purpose) {
                            errors.push(`Behavior ${index + 1}: Invoke action missing 'purpose'`);
                        }
                    }
                }

                // Validate QoS
                if (behavior.qos !== undefined && ![0, 1, 2].includes(behavior.qos)) {
                    errors.push(`Behavior ${index + 1}: Invalid QoS level ${behavior.qos}. Must be 0, 1, or 2`);
                }
            });
        }

        // Validate prompt if present
        if (data.prompt) {
            if (!data.prompt.mode || !["supplement", "replace"].includes(data.prompt.mode)) {
                errors.push("Invalid prompt.mode. Must be 'supplement' or 'replace'");
            }
            if (!data.prompt.source || !["direct", "resource"].includes(data.prompt.source)) {
                errors.push("Invalid prompt.source. Must be 'direct' or 'resource'");
            }
            if (data.prompt.source === "direct" && !data.prompt.content) {
                errors.push("Direct prompt missing 'content'");
            }
            if (data.prompt.source === "resource" && !data.prompt.resourceId) {
                errors.push("Resource prompt missing 'resourceId'");
            }
        }

        return {
            valid: errors.length === 0,
            errors,
        };
    }

    private convertAgentToResource(agent: AgentValidationData): ResourceCreateInput {
        return {
            id: agent.identity.id || generatePK().toString(),
            publicId: `agent-${agent.identity.name.toLowerCase().replace(/\s+/g, "-")}`,
            resourceType: ResourceType.Routine,
            isPrivate: false,
            versionsCreate: [{
                id: generatePK().toString(),
                resourceSubType: ResourceSubType.RoutineInformational,
                config: { 
                    ...agent, 
                    __version: "1.0",
                    resources: agent.resources?.map(link => ({ 
                        link, 
                        usedFor: ResourceUsedFor.Context,
                        translations: [{ language: "en", name: link }],
                    })) || [],
                },
                translationsCreate: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: agent.identity.name,
                    description: agent.goal,
                }],
                isPrivate: false,
                versionLabel: "1.0.0",
            }],
            isInternal: false,
        };
    }

    private convertResourceToAgent(resource: Resource): AgentValidationData {
        const version = resource.versions?.[0];
        if (!version || !version.config) {
            throw new Error("Invalid agent resource: missing version or config");
        }
        return version.config as unknown as AgentValidationData;
    }
}
