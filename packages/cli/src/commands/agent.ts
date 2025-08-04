import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import chalk from "chalk";
import { promises as fs } from "fs";
import * as path from "path";
import { glob } from "glob";
import * as cliProgress from "cli-progress";
import { UI } from "../utils/constants.js";
import type { 
    Resource,
    ResourceVersionSearchInput,
    ResourceVersionSearchResult,
    BotConfigObject,
    ResourceCreateInput,
} from "@vrooli/shared";
import { generatePK, ResourceType, ResourceSubType, ResourceUsedFor, endpointsResource, type ResourceVersionEdge } from "@vrooli/shared";
import { BaseCommand } from "./BaseCommand.js";

// Interface definitions for agent operations
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

export class AgentCommands extends BaseCommand {
    constructor(
        program: Command,
        client: ApiClient,
        config: ConfigManager,
    ) {
        super(program, client, config);
    }

    protected registerCommands(): void {
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
        try {
            // Read and parse file
            const agentData = await this.executeWithSpinner(
                "Reading and parsing agent file...",
                async () => this.parseJsonFile<AgentValidationData>(filePath),
                "File parsed successfully",
            );

            // Validate structure
            const validation = await this.executeWithSpinner(
                "Validating agent structure...",
                async () => this.validateAgentData(agentData, options.validate),
                "Validation completed",
            );
            
            if (!validation.valid) {
                const errorMessage = `Validation errors:\n${validation.errors.map(error => `  - ${error}`).join("\n")}`;
                this.handleError(new Error(errorMessage));
            }

            if (options.dryRun) {
                output.success("Agent is valid (dry run)");
                output.info(`  Name: ${agentData.identity.name}`);
                output.info(`  Goal: ${agentData.goal}`);
                output.info(`  Role: ${agentData.role}`);
                output.info(`  Subscriptions: ${agentData.subscriptions.length}`);
                output.info(`  Behaviors: ${agentData.behaviors.length}`);
                return;
            }

            // Convert agent format to Resource format for API
            const resourceData = this.convertAgentToResource(agentData);
            const response = await this.executeWithSpinner(
                "Uploading to server...",
                async () => this.client.post<Resource>("/api/resource", resourceData),
                "Agent imported successfully",
            );

            this.output(
                response,
                () => {
                    output.success("Import successful");
                    output.info(`  ID: ${response.id}`);
                    output.info(`  Name: ${agentData.identity.name}`);
                    output.info(`  Role: ${agentData.role}`);
                    output.info(`  URL: ${this.config.getServerUrl()}/agent/${response.id}`);
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
            const pattern = path.join(absolutePath, "**", options.pattern || "*.json");
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
                const errorMessage = `${results.failed.length} file(s) failed to import:\n${results.failed.map(f => `  - ${f.file}: ${f.error}`).join("\n")}`;
                this.handleError(new Error(errorMessage));
            }
        } catch (error) {
            this.handleError(error, "Directory import failed");
        }
    }

    private async exportAgent(agentId: string, options: { output?: string }): Promise<void> {
        try {
            const agent = await this.executeWithSpinner(
                "Fetching agent...",
                async () => this.client.get<Resource>(`/api/resource/${agentId}`),
                "Agent fetched",
            );

            // Convert back to agent format
            const agentData = this.convertResourceToAgent(agent);

            // Determine output path
            const outputPath = options.output || `agent-${agentData.identity.name}.json`;
            const absolutePath = path.resolve(outputPath);

            // Write to file
            await this.writeJsonFile(absolutePath, agentData);

            output.success(`Agent exported to: ${absolutePath}`);
        } catch (error) {
            this.handleError(error, "Export failed");
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
                resourceSubType: ResourceSubType.RoutineInformational,
            };

            if (options.search) {
                params.searchString = options.search;
            }

            if (options.mine) {
                params.createdByIdRoot = "me";
            }

            // Note: Role filtering would need to be done post-fetch since it's in metadata
            const response = await this.client.get<ResourceVersionSearchResult>("/api/resources", { params });

            if (options.format === "json") {
                output.json(response);
                return;
            }

            if (!response || !response.edges || response.edges.length === 0) {
                output.info(chalk.yellow("No agents found"));
                return;
            }

            // Filter by role if specified
            let filteredEdges = response.edges;
            if (options.role) {
                filteredEdges = response.edges.filter(edge => {
                    const metadata = edge.node.config as BotConfigObject;
                    const role = metadata?.agentSpec?.role || "unknown";
                    return role === options.role;
                });
            }

            this.output(
                { edges: filteredEdges, count: filteredEdges.length },
                () => {
                    output.info(chalk.bold(`\nAgents found (${filteredEdges.length}):\n`));

                    filteredEdges.forEach((edge, index: number) => {
                        const resource = edge.node;
                        const metadata = resource.config as BotConfigObject;
                        const role = metadata?.agentSpec?.role || "unknown";

                        output.info(chalk.cyan(`${index + 1}. ${resource.translations?.[0]?.name || "Unnamed Agent"}`));
                        output.info(`   ID: ${resource.id}`);
                        output.info(`   Role: ${role}`);
                        output.info(`   Goal: ${metadata?.agentSpec?.goal || "No goal specified"}`);
                        output.info(`   Created: ${new Date(resource.createdAt).toLocaleDateString()}`);
                        output.info("");
                    });

                    if (response.pageInfo.hasNextPage) {
                        output.info(chalk.gray("More results available. Use pagination to see more."));
                    }
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to list agents");
        }
    }

    private async getAgent(agentId: string): Promise<void> {
        try {
            const agent = await this.executeWithSpinner(
                "Fetching agent...",
                async () => this.client.get<Resource>(`/api/resource/${agentId}`),
                "Agent fetched",
            );

            // Extract agent data from resource
            const agentData = this.convertResourceToAgent(agent);

            this.output(
                agent,
                () => {
                    output.info(chalk.bold("\nAgent Details:"));
                    output.info(`  ID: ${agent.id}`);
                    output.info(`  Name: ${agentData.identity.name}`);
                    output.info(`  Goal: ${agentData.goal}`);
                    output.info(`  Role: ${agentData.role}`);
                    output.info(`  Created: ${agent.createdAt ? new Date(agent.createdAt).toLocaleString() : "Unknown"}`);
                    output.info(`  Updated: ${agent.updatedAt ? new Date(agent.updatedAt).toLocaleString() : "Unknown"}`);
                    
                    output.info(chalk.bold("\nSubscriptions:"));
                    agentData.subscriptions.forEach((sub: string, index: number) => {
                        output.info(`  ${index + 1}. ${sub}`);
                    });

                    output.info(chalk.bold("\nBehaviors:"));
                    agentData.behaviors.forEach((behavior, index: number) => {
                        output.info(`  ${index + 1}. Trigger: ${behavior.trigger.topic}`);
                        if (behavior.trigger.when) {
                            output.info(`     When: ${behavior.trigger.when}`);
                        }
                        output.info(`     Action: ${behavior.action.type}`);
                        if (behavior.action.type === "routine") {
                            output.info(`     Routine: ${behavior.action.label}`);
                        } else {
                            output.info(`     Purpose: ${behavior.action.purpose}`);
                        }
                    });

                    if (agentData.prompt) {
                        output.info(chalk.bold("\nPrompt:"));
                        output.info(`  Mode: ${agentData.prompt.mode}`);
                        output.info(`  Source: ${agentData.prompt.source}`);
                        if (agentData.prompt.content) {
                            output.info(`  Content: ${agentData.prompt.content.substring(0, 100)}...`);
                        }
                    }
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to fetch agent");
        }
    }

    private async validateAgent(filePath: string, options: { checkRoutines?: boolean }): Promise<void> {
        output.info(chalk.bold("🔍 Validating agent file...\n"));

        try {
            // Parse JSON
            const agentData = await this.parseJsonFile<AgentValidationData>(filePath);

            // Perform validation
            const validation = await this.validateAgentData(agentData, options.checkRoutines);

            if (validation.valid) {
                output.success("Agent is valid!");
                output.info(`\n  Name: ${agentData.identity.name}`);
                output.info(`  Goal: ${agentData.goal}`);
                output.info(`  Role: ${agentData.role}`);
                output.info(`  Subscriptions: ${agentData.subscriptions.length}`);
                output.info(`  Behaviors: ${agentData.behaviors.length}`);
            } else {
                const errorMessage = `Validation failed:\n${validation.errors.map(error => `  - ${error}`).join("\n")}`;
                this.handleError(new Error(errorMessage));
            }
        } catch (error) {
            this.handleError(error, "Validation error");
        }
    }

    private async searchAgents(query: string, options: {
        limit?: string;
        role?: string;
        format?: string;
    }): Promise<void> {
        try {
            const searchInput: ResourceVersionSearchInput = {
                take: parseInt(options.limit || "10"),
                searchString: query,
                isLatest: true,
                rootResourceType: ResourceType.Routine,
            };

            const response = await this.executeWithSpinner(
                `Searching for agents: "${query}"...`,
                async () => this.client.requestWithEndpoint<ResourceVersionSearchResult>(
                    endpointsResource.findMany,
                    searchInput as Record<string, unknown>,
                ),
                "Search complete",
            );

            if (!response.edges || response.edges.length === 0) {
                output.info(`No agents found for "${query}"`);
                return;
            }

            output.info(`Found ${response.edges.length} agent(s) for "${query}"`);

            const agents = response.edges.map((edge: ResourceVersionEdge) => {
                const translation = edge.node.translations?.[0];
                return {
                    id: edge.node.id,
                    publicId: edge.node.publicId,
                    name: translation?.name || "Unnamed",
                    description: translation?.description || "No description",
                    score: 1.0, // searchScore not available in current API
                };
            });

            if (options.format === "json") {
                output.json(agents);
            } else {
                const tableData = agents.map((agent) => ({
                    Score: agent.score.toFixed(2),
                    ID: agent.publicId,
                    Name: agent.name.substring(0, UI.PADDING.BOTTOM - 2),
                    Description: agent.description.substring(0, UI.DESCRIPTION_WIDTH),
                }));
                output.table(tableData);
            }
        } catch (error) {
            this.handleError(error, "Failed to search agents");
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
                            output.info(chalk.gray(`  Checking routine '${behavior.action.label}'...`));
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
