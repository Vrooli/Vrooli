import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import chalk from "chalk";
import { promises as fs } from "fs";
import * as path from "path";
import { UI } from "../utils/constants.js";
import type { 
    Team,
    TeamCreateInput,
    TeamSearchInput,
    TeamSearchResult,
    Chat,
    ChatCreateInput,
    ResourceQuota,
    BlackboardItem,
} from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";
import { BaseCommand } from "./BaseCommand.js";

// Interface for blackboard insights
interface TeamInsight {
    type: string;
    confidence: number;
    data: unknown;
}

// Constants for team operations
const TEAM_CONSTANTS = {
    MICROSECONDS_PER_SECOND: 1000000,
    TIME_WINDOW_SECONDS: 60,
    POLLING_INTERVAL_MS: 1000,
    THROTTLE_THRESHOLD: 0.8,
    MEMORY_USAGE_WARNING: 0.6,
    SYSTEM_CHECK_INTERVAL: 60,
    DEFAULT_RESOURCE_LIMIT: 20,
    TABLE_WIDTH: 80,
    COLUMN_WIDTH: 50,
    HISTORY_WINDOW_MINUTES: 5,
    LOG_MESSAGES_LIMIT: 200,
    TIME_OFFSET_HOURS: 3,
} as const;

// Interface definitions for team operations

interface TeamValidationData {
    __version: string;
    deploymentType: "development" | "saas" | "appliance";
    goal: string;
    businessPrompt: string;
    resourceQuota: ResourceQuota;
    targetProfitPerMonth: string;
    costLimit?: string;
    verticalPackage?: {
        industry: string;
        complianceRequirements?: string[];
        defaultWorkflows?: string[];
        dataPrivacyLevel: "standard" | "hipaa" | "pci" | "sox" | "gdpr";
        terminology?: Record<string, string>;
        regulatoryBodies?: string[];
    };
    marketSegment?: "enterprise" | "smb" | "consumer";
    isolation?: {
        sandboxLevel: "none" | "user-namespace" | "full-container" | "vm";
        networkPolicy: "open" | "restricted" | "isolated" | "air-gapped";
        secretsAccess: string[];
        auditLogging?: boolean;
        securityPolicies?: string[];
    };
    preferredModel?: string;
    stats: {
        totalInstances: number;
        totalProfit: string;
        totalCosts: string;
        averageKPIs: Record<string, number>;
        activeInstances: number;
        lastUpdated: number;
    };
    blackboard?: BlackboardItem[];
    defaultLimits?: {
        maxToolCallsPerBotResponse: number;
        maxToolCalls: number;
        maxCreditsPerBotResponse: string;
        maxCredits: string;
        maxDurationPerBotResponseMs: number;
        maxDurationMs: number;
        delayBetweenProcessingCyclesMs: number;
    };
    defaultScheduling?: {
        defaultDelayMs: number;
        toolSpecificDelays: Record<string, number>;
        requiresApprovalTools: string[] | "all" | "none";
        approvalTimeoutMs: number;
        autoRejectOnTimeout: boolean;
    };
    defaultPolicy?: {
        visibility: "open" | "restricted" | "private";
        acl?: string[];
    };
}

interface _TeamSearchEdge {
    node: Team;
    searchScore?: number;
}

interface _TeamMonitorData {
    id: string;
    name: string;
    deploymentType: string;
    goal: string;
    stats: {
        totalInstances: number;
        activeInstances: number;
        totalProfitUSD: number;
        totalCostsUSD: number;
        netProfitUSD: number;
        profitPerInstance: number;
        isAboveTarget: boolean;
        isWithinCostLimit: boolean;
        targetAchievementRatio: number;
    };
    resourceUtilization: {
        gpuPercentage: number;
        ramGB: number;
        cpuCores: number;
        storageGB: number;
    };
    recentInsights: Array<{
        type: string;
        data: unknown;
        confidence: number;
        created: string;
    }>;
}

export class TeamCommands extends BaseCommand {
    constructor(
        program: Command,
        client: ApiClient,
        config: ConfigManager,
    ) {
        super(program, client, config);
    }

    protected registerCommands(): void {
        const teamCmd = this.program
            .command("team")
            .description("Manage teams for swarm orchestration");

        teamCmd
            .command("create")
            .description("Create a new team from configuration")
            .option("-c, --config <file>", "Team configuration JSON file")
            .option("-n, --name <name>", "Team name")
            .option("-g, --goal <goal>", "Team goal")
            .option("-t, --type <type>", "Deployment type (development|saas|appliance)", "development")
            .option("--gpu <percent>", "GPU percentage allocation", String(TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT))
            .option("--ram <gb>", "RAM allocation in GB", "16")
            .option("--target-profit <amount>", "Target monthly profit in USD", "2500")
            .action(async (options) => {
                await this.createTeam(options);
            });

        teamCmd
            .command("list")
            .description("List teams")
            .option("-l, --limit <limit>", "Number of teams to show", String(TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT))
            .option("-s, --search <query>", "Search teams by name or goal")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .option("--mine", "Show only my teams")
            .option("-t, --type <type>", "Filter by deployment type")
            .action(async (options) => {
                await this.listTeams(options);
            });

        teamCmd
            .command("get <teamId>")
            .description("Get team details")
            .option("--show-blackboard", "Include blackboard insights", false)
            .option("--show-stats", "Include detailed statistics", false)
            .action(async (teamId, options) => {
                await this.getTeam(teamId, options);
            });

        teamCmd
            .command("monitor <teamId>")
            .description("Monitor team performance and resource usage")
            .option("--interval <seconds>", "Update interval in seconds", "5")
            .option("--duration <minutes>", "Monitoring duration in minutes", "0")
            .option("--show-blackboard", "Show blackboard insights", true)
            .option("--show-stats", "Show detailed statistics", true)
            .action(async (teamId, options) => {
                await this.monitorTeam(teamId, options);
            });

        teamCmd
            .command("spawn <teamId>")
            .description("Spawn a new chat instance from team template")
            .option("-n, --name <name>", "Name for the chat instance")
            .option("-c, --context <json>", "Client context as JSON")
            .option("-t, --task <task>", "Specific task for this instance")
            .option("--auto-start", "Automatically start the chat", false)
            .action(async (teamId, options) => {
                await this.spawnChat(teamId, options);
            });

        teamCmd
            .command("update <teamId>")
            .description("Update team configuration")
            .option("--goal <goal>", "Update team goal")
            .option("--prompt <prompt>", "Update business prompt")
            .option("--target-profit <amount>", "Update target profit in USD")
            .option("--cost-limit <amount>", "Update cost limit in USD")
            .action(async (teamId, options) => {
                await this.updateTeam(teamId, options);
            });

        teamCmd
            .command("import <file>")
            .description("Import team configuration from JSON file")
            .option("--dry-run", "Validate without creating")
            .action(async (file, options) => {
                await this.importTeam(file, options);
            });

        teamCmd
            .command("export <teamId>")
            .description("Export team configuration to JSON file")
            .option("-o, --output <file>", "Output file path")
            .action(async (teamId, options) => {
                await this.exportTeam(teamId, options);
            });

        teamCmd
            .command("insights <teamId>")
            .description("View team blackboard insights")
            .option("-t, --type <type>", "Filter by insight type")
            .option("-l, --limit <limit>", "Number of insights to show", String(TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT))
            .option("--min-confidence <score>", "Minimum confidence score", "0.7")
            .action(async (teamId, options) => {
                await this.viewInsights(teamId, options);
            });
    }

    private async createTeam(options: {
        config?: string;
        name?: string;
        goal?: string;
        type?: string;
        gpu?: string;
        ram?: string;
        targetProfit?: string;
    }): Promise<void> {
        try {
            let teamConfig: TeamValidationData;

            if (options.config) {
                // Load from config file
                teamConfig = await this.parseJsonFile<TeamValidationData>(options.config);
            } else {
                // Create from command line options
                if (!options.name || !options.goal) {
                    output.error("Name and goal are required when not using config file");
                    output.error("Use --name and --goal or provide --config file");
                    process.exit(1);
                }

                const targetProfitMicroDollars = parseInt(options.targetProfit || "2500") * TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;

                teamConfig = {
                    __version: "1.0",
                    deploymentType: (options.type as "development" | "saas" | "appliance") || "development",
                    goal: options.goal,
                    businessPrompt: `You are a ${options.name} team focused on: ${options.goal}. Optimize for profitability while maintaining quality.`,
                    resourceQuota: {
                        gpuPercentage: parseInt(options.gpu || String(TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT)),
                        ramGB: parseInt(options.ram || "16"),
                        cpuCores: 4,
                        storageGB: 100,
                    },
                    targetProfitPerMonth: targetProfitMicroDollars.toString(),
                    stats: {
                        totalInstances: 0,
                        totalProfit: "0",
                        totalCosts: "0",
                        averageKPIs: {},
                        activeInstances: 0,
                        lastUpdated: Date.now(),
                    },
                };
            }

            // Validate team configuration
            const validation = this.validateTeamConfig(teamConfig);
            if (!validation.valid) {
                output.error("Invalid team configuration:");
                validation.errors.forEach((error: string) => {
                    output.error(`  - ${error}`);
                });
                process.exit(1);
            }

            // Convert to TeamCreateInput
            const teamInput: TeamCreateInput = {
                id: generatePK().toString(),
                isPrivate: true,
                config: teamConfig,
                translationsCreate: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: options.name || "Unnamed Team",
                    bio: teamConfig.goal,
                }],
            };

            const result = await this.executeWithSpinner(
                "Creating team...",
                async () => this.client.post<Team>("/team", teamInput),
                "Team created successfully",
            );

            this.output(
                result,
                () => {
                    output.success("Team created");
                    output.info(`  ID: ${result.id}`);
                    output.info(`  Name: ${options.name || "Unnamed Team"}`);
                    output.info(`  Goal: ${teamConfig.goal}`);
                    output.info(`  Type: ${teamConfig.deploymentType}`);
                    output.info(`  Target Profit: $${parseInt(options.targetProfit || "2500")}/month`);
                    output.info(chalk.gray(`\nUse 'vrooli team spawn ${result.id}' to create chat instances`));
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to create team");
        }
    }

    private async listTeams(options: {
        limit?: string;
        search?: string;
        format?: string;
        mine?: boolean;
        type?: string;
    }): Promise<void> {
        try {
            const searchInput: TeamSearchInput = {
                take: parseInt(options.limit || String(TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT)),
                searchString: options.search,
                isOpenToNewMembers: false, // Teams are templates, not open for members
            };

            // Note: TeamSearchInput doesn't support filtering by creator
            // We'll need to filter results post-fetch if needed

            const result = await this.executeWithSpinner(
                "Fetching teams...",
                async () => this.client.post<TeamSearchResult>("/teams", searchInput),
                "Teams fetched",
            );

            if (options.format === "json") {
                output.json(result);
                return;
            }

            if (!result || !result.edges || result.edges.length === 0) {
                output.info(chalk.yellow("No teams found"));
                return;
            }

            // Filter by deployment type if specified
            let teams = result.edges.map(edge => edge.node);
            if (options.type) {
                teams = teams.filter(team => {
                    const config = team.config as TeamValidationData;
                    return config?.deploymentType === options.type;
                });
            }

            this.output(
                { teams, count: teams.length },
                () => {
                    output.info(`Found ${teams.length} teams`);
                    this.displayTeamsTable(teams);
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to list teams");
        }
    }

    private async getTeam(teamId: string, options: {
        showBlackboard?: boolean;
        showStats?: boolean;
    }): Promise<void> {
        try {
            const team = await this.executeWithSpinner(
                "Fetching team...",
                async () => this.client.get<Team>(`/team/${teamId}`),
                "Team fetched",
            );

            this.output(
                team,
                () => {
                    const config = team.config as TeamValidationData;
                    const name = team.translations?.[0]?.name || "Unnamed Team";

                    output.info(chalk.bold("\nTeam Details:"));
                    output.info(`  ID: ${team.id}`);
                    output.info(`  Name: ${name}`);
                    output.info(`  Goal: ${config.goal}`);
                    output.info(`  Deployment Type: ${config.deploymentType}`);
                    output.info(`  Created: ${team.createdAt ? new Date(team.createdAt).toLocaleString() : "Unknown"}`);
                    output.info(`  Updated: ${team.updatedAt ? new Date(team.updatedAt).toLocaleString() : "Unknown"}`);

                    output.info(chalk.bold("\nResource Allocation:"));
                    output.info(`  GPU: ${config.resourceQuota.gpuPercentage}%`);
                    output.info(`  RAM: ${config.resourceQuota.ramGB} GB`);
                    output.info(`  CPU: ${config.resourceQuota.cpuCores} cores`);
                    output.info(`  Storage: ${config.resourceQuota.storageGB} GB`);

                    output.info(chalk.bold("\nEconomic Targets:"));
                    const targetProfitUSD = Number(BigInt(config.targetProfitPerMonth)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                    output.info(`  Target Profit: $${targetProfitUSD}/month`);
                    if (config.costLimit) {
                        const costLimitUSD = Number(BigInt(config.costLimit)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                        output.info(`  Cost Limit: $${costLimitUSD}/month`);
                    }

                    if (options.showStats && config.stats) {
                        output.info(chalk.bold("\nPerformance Statistics:"));
                        const totalProfitUSD = Number(BigInt(config.stats.totalProfit)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                        const totalCostsUSD = Number(BigInt(config.stats.totalCosts)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                        const netProfitUSD = totalProfitUSD - totalCostsUSD;
                        
                        output.info(`  Total Instances: ${config.stats.totalInstances}`);
                        output.info(`  Active Instances: ${config.stats.activeInstances}`);
                        output.info(`  Total Revenue: $${totalProfitUSD.toFixed(2)}`);
                        output.info(`  Total Costs: $${totalCostsUSD.toFixed(2)}`);
                        output.info(`  Net Profit: $${netProfitUSD.toFixed(2)}`);
                        
                        if (config.stats.totalInstances > 0) {
                            output.info(`  Profit/Instance: $${(netProfitUSD / config.stats.totalInstances).toFixed(2)}`);
                        }
                        
                        const targetRatio = targetProfitUSD > 0 ? (netProfitUSD / targetProfitUSD * 100).toFixed(1) : "N/A";
                        output.info(`  Target Achievement: ${targetRatio}%`);
                    }

                    if (options.showBlackboard && config.blackboard && config.blackboard.length > 0) {
                        output.info(chalk.bold("\nRecent Blackboard Insights:"));
                        const recentInsights = config.blackboard.slice(-TEAM_CONSTANTS.HISTORY_WINDOW_MINUTES);
                        recentInsights.forEach((item, index) => {
                            const insight = item.value as TeamInsight;
                            output.info(`  ${index + 1}. ${insight.type} (confidence: ${insight.confidence})`);
                            output.info(`     ${JSON.stringify(insight.data).substring(0, 100)}...`);
                            output.info(`     ${new Date(item.created_at).toLocaleString()}`);
                        });
                    }

                    output.info(chalk.bold("\nBusiness Prompt:"));
                    output.info(`  ${config.businessPrompt.substring(0, TEAM_CONSTANTS.LOG_MESSAGES_LIMIT)}...`);
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to fetch team");
        }
    }

    private async monitorTeam(teamId: string, options: {
        interval?: string;
        duration?: string;
        showBlackboard?: boolean;
        showStats?: boolean;
    }): Promise<void> {
        const intervalMs = parseInt(options.interval || "5") * TEAM_CONSTANTS.POLLING_INTERVAL_MS;
        const durationMs = parseInt(options.duration || "0") * TEAM_CONSTANTS.TIME_WINDOW_SECONDS * TEAM_CONSTANTS.POLLING_INTERVAL_MS;
        const endTime = durationMs > 0 ? Date.now() + durationMs : 0;

        output.info(chalk.cyan(`📊 Monitoring team ${teamId}...`));
        output.info(chalk.gray("Press Ctrl+C to stop\n"));

        const monitor = async () => {
            try {
                const team = await this.client.get<Team>(`/team/${teamId}`);
                const config = team.config as TeamValidationData;
                const name = team.translations?.[0]?.name || "Unnamed Team";

                // Clear console for clean display
                console.clear();
                output.info(chalk.cyan(`📊 Team Monitor: ${name}`));
                output.info(chalk.gray(`Updated: ${new Date().toLocaleTimeString()}`));
                output.info("─".repeat(UI.TERMINAL_WIDTH));

                // Basic info
                output.info(chalk.bold("\n🎯 Goal:") + " " + config.goal);
                output.info(chalk.bold("📍 Type:") + " " + config.deploymentType);

                // Performance metrics
                if (options.showStats && config.stats) {
                    const totalProfitUSD = Number(BigInt(config.stats.totalProfit)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                    const totalCostsUSD = Number(BigInt(config.stats.totalCosts)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                    const netProfitUSD = totalProfitUSD - totalCostsUSD;
                    const targetProfitUSD = Number(BigInt(config.targetProfitPerMonth)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                    const targetRatio = targetProfitUSD > 0 ? (netProfitUSD / targetProfitUSD * 100) : 0;

                    output.info(chalk.bold("\n📈 Performance:"));
                    output.info(`  Instances: ${chalk.green(config.stats.activeInstances)} active / ${config.stats.totalInstances} total`);
                    output.info(`  Revenue: ${chalk.green(`$${totalProfitUSD.toFixed(2)}`)}`);
                    output.info(`  Costs: ${chalk.red(`$${totalCostsUSD.toFixed(2)}`)}`);
                    output.info(`  Net Profit: ${netProfitUSD >= 0 ? chalk.green(`$${netProfitUSD.toFixed(2)}`) : chalk.red(`$${netProfitUSD.toFixed(2)}`)}`);
                    
                    // Progress bar for target achievement
                    const progressBar = this.createProgressBar(targetRatio, 100);
                    output.info(`  Target Progress: ${progressBar} ${targetRatio.toFixed(1)}%`);

                    // Resource utilization
                    output.info(chalk.bold("\n💻 Resource Usage:"));
                    const gpuBar = this.createProgressBar(config.resourceQuota.gpuPercentage, 100);
                    output.info(`  GPU: ${gpuBar} ${config.resourceQuota.gpuPercentage}%`);
                    output.info(`  RAM: ${config.resourceQuota.ramGB} GB | CPU: ${config.resourceQuota.cpuCores} cores`);
                }

                // Recent insights
                if (options.showBlackboard && config.blackboard && config.blackboard.length > 0) {
                    output.info(chalk.bold("\n🧠 Recent Insights:"));
                    const recentInsights = config.blackboard.slice(-TEAM_CONSTANTS.TIME_OFFSET_HOURS);
                    recentInsights.forEach((item) => {
                        const insight = item.value as TeamInsight;
                        const icon = insight.confidence > TEAM_CONSTANTS.THROTTLE_THRESHOLD ? "🟢" : insight.confidence > TEAM_CONSTANTS.MEMORY_USAGE_WARNING ? "🟡" : "🔴";
                        output.info(`  ${icon} ${insight.type}: ${JSON.stringify(insight.data).substring(0, TEAM_CONSTANTS.SYSTEM_CHECK_INTERVAL)}...`);
                    });
                }

                // KPIs if available
                if (config.stats.averageKPIs && Object.keys(config.stats.averageKPIs).length > 0) {
                    output.info(chalk.bold("\n📊 Key Performance Indicators:"));
                    Object.entries(config.stats.averageKPIs).slice(0, TEAM_CONSTANTS.HISTORY_WINDOW_MINUTES).forEach(([key, value]) => {
                        output.info(`  ${key}: ${typeof value === "number" ? value.toFixed(2) : value}`);
                    });
                }

                output.info("\n" + "─".repeat(UI.TERMINAL_WIDTH));

                // Check if we should continue
                if (endTime > 0 && Date.now() >= endTime) {
                    output.success("Monitoring complete");
                    process.exit(0);
                }
            } catch (error) {
                output.error(`Monitor error: ${error}`);
            }
        };

        // Initial display
        await monitor();

        // Set up interval
        const intervalId = setInterval(monitor, intervalMs);

        // Handle graceful shutdown
        process.on("SIGINT", () => {
            clearInterval(intervalId);
            output.info(chalk.yellow("\n\n⚠ Monitoring stopped"));
            process.exit(0);
        });
    }

    private async spawnChat(teamId: string, options: {
        name?: string;
        context?: string;
        task?: string;
        autoStart?: boolean;
    }): Promise<void> {
        try {
            // Fetch team to get configuration
            const team = await this.executeWithSpinner(
                "Fetching team configuration...",
                async () => this.client.get<Team>(`/team/${teamId}`),
                "Team configuration fetched",
            );
            
            const teamConfig = team.config as TeamValidationData;
            const teamName = team.translations?.[0]?.name || "Unnamed Team";

            // Parse client context if provided
            let _clientContext = {};
            if (options.context) {
                try {
                    _clientContext = JSON.parse(options.context);
                } catch (error) {
                    output.error("Invalid client context JSON");
                    process.exit(1);
                }
            }

            // Create chat with team configuration
            const chatInput: ChatCreateInput = {
                id: generatePK().toString(),
                teamConnect: teamId,
                openToAnyoneWithInvite: teamConfig.defaultPolicy?.visibility === "open",
                task: options.task || teamConfig.goal,
                translationsCreate: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: options.name || `${teamName} - ${new Date().toLocaleDateString()}`,
                }],
            };

            const result = await this.executeWithSpinner(
                "Creating chat instance...",
                async () => this.client.post<Chat>("/chat", chatInput),
                "Chat instance spawned successfully",
            );

            this.output(
                result,
                () => {
                    output.success("Chat instance created");
                    output.info(`  Chat ID: ${result.id}`);
                    output.info(`  Name: ${result.translations?.[0]?.name}`);
                    output.info(`  Team: ${teamName}`);
                    output.info(`  Task: ${options.task || teamConfig.goal}`);
                    
                    if (options.autoStart) {
                        output.info(chalk.gray("\n🚀 Starting interactive chat..."));
                        // Could trigger interactive chat here
                        output.info(chalk.cyan(`vrooli chat interactive ${result.id}`));
                    } else {
                        output.info(chalk.gray(`\nUse 'vrooli chat interactive ${result.id}' to start chatting`));
                    }
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to spawn chat");
        }
    }

    private async updateTeam(teamId: string, options: {
        goal?: string;
        prompt?: string;
        targetProfit?: string;
        costLimit?: string;
    }): Promise<void> {
        try {
            // Fetch current team and build updates
            const team = await this.executeWithSpinner(
                "Fetching current team...",
                async () => this.client.get<Team>(`/team/${teamId}`),
                "Team fetched",
            );
            
            const currentConfig = team.config as TeamValidationData;

            // Build update payload
            const updates: Record<string, unknown> = {};
            
            if (options.goal) {
                updates.goal = options.goal;
            }
            
            if (options.prompt) {
                updates.businessPrompt = options.prompt;
            }
            
            if (options.targetProfit) {
                const targetProfitMicroDollars = parseInt(options.targetProfit) * TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                updates.targetProfitPerMonth = targetProfitMicroDollars.toString();
            }
            
            if (options.costLimit) {
                const costLimitMicroDollars = parseInt(options.costLimit) * TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                updates.costLimit = costLimitMicroDollars.toString();
            }

            // Merge with existing config
            const updatedConfig = { ...currentConfig, ...updates };

            // Update via API
            const result = await this.executeWithSpinner(
                "Updating team...",
                async () => this.client.put<Team>(`/team/${teamId}`, {
                    config: updatedConfig,
                }),
                "Team updated successfully",
            );

            this.output(
                result,
                () => {
                    output.success("Team updated");
                    if (options.goal) output.info(`  Goal: ${options.goal}`);
                    if (options.targetProfit) output.info(`  Target Profit: $${options.targetProfit}/month`);
                    if (options.costLimit) output.info(`  Cost Limit: $${options.costLimit}/month`);
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to update team");
        }
    }

    private async importTeam(filePath: string, options: { dryRun?: boolean }): Promise<void> {
        try {
            // Read and parse team configuration
            const teamConfig = await this.executeWithSpinner(
                "Reading and parsing team configuration...",
                async () => {
                    const absolutePath = path.resolve(filePath);
                    const fileContent = await fs.readFile(absolutePath, "utf-8");
                    return JSON.parse(fileContent);
                },
                "Configuration parsed",
            );

            // Validate configuration
            const validation = await this.executeWithSpinner(
                "Validating team configuration...",
                async () => this.validateTeamConfig(teamConfig),
                "Validation completed",
            );
            
            if (!validation.valid) {
                output.error("Validation failed:");
                validation.errors.forEach((error: string) => {
                    output.error(`  - ${error}`);
                });
                process.exit(1);
            }

            if (options.dryRun) {
                output.success("Team configuration is valid (dry run)");
                const targetProfitUSD = Number(BigInt(teamConfig.targetProfitPerMonth)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
                output.info(`  Deployment Type: ${teamConfig.deploymentType}`);
                output.info(`  Goal: ${teamConfig.goal}`);
                output.info(`  Target Profit: $${targetProfitUSD}/month`);
                output.info(`  Resource Quota: ${teamConfig.resourceQuota.gpuPercentage}% GPU, ${teamConfig.resourceQuota.ramGB}GB RAM`);
                return;
            }

            // Create team via API
            const teamInput: TeamCreateInput = {
                id: generatePK().toString(),
                isPrivate: true,
                config: teamConfig,
                translationsCreate: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: `Imported Team - ${new Date().toISOString()}`,
                    bio: teamConfig.goal,
                }],
            };

            const result = await this.executeWithSpinner(
                "Creating team...",
                async () => this.client.post<Team>("/team", teamInput),
                "Team imported successfully",
            );

            this.output(
                result,
                () => {
                    output.success("Import successful");
                    output.info(`  Team ID: ${result.id}`);
                    output.info(`  Goal: ${teamConfig.goal}`);
                },
            );
        } catch (error) {
            this.handleError(error, "Import failed");
        }
    }

    private async exportTeam(teamId: string, options: { output?: string }): Promise<void> {
        try {
            const team = await this.executeWithSpinner(
                "Fetching team configuration...",
                async () => this.client.get<Team>(`/team/${teamId}`),
                "Team fetched",
            );

            const teamConfig = team.config as TeamValidationData;
            
            // Determine output path
            const outputPath = options.output || `team-${teamId}-config.json`;
            const absolutePath = path.resolve(outputPath);

            // Write to file  
            await this.writeJsonFile(absolutePath, teamConfig);

            output.success(`Team configuration exported to: ${absolutePath}`);
        } catch (error) {
            this.handleError(error, "Export failed");
        }
    }

    private async viewInsights(teamId: string, options: {
        type?: string;
        limit?: string;
        minConfidence?: string;
    }): Promise<void> {
        try {
            const team = await this.executeWithSpinner(
                "Fetching team insights...",
                async () => this.client.get<Team>(`/team/${teamId}`),
                "Insights fetched",
            );
            
            const config = team.config as TeamValidationData;

            if (!config.blackboard || config.blackboard.length === 0) {
                output.info(chalk.yellow("No insights found in team blackboard"));
                return;
            }

            // Filter insights
            let insights = config.blackboard;
            const minConfidence = parseFloat(options.minConfidence || "0.7");

            insights = insights.filter(item => {
                const insight = item.value as TeamInsight;
                
                // Type filter
                if (options.type && insight.type !== options.type) {
                    return false;
                }
                
                // Confidence filter
                if (insight.confidence < minConfidence) {
                    return false;
                }
                
                return true;
            });

            // Limit results
            const limit = parseInt(options.limit || String(TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT));
            insights = insights.slice(-limit);

            if (insights.length === 0) {
                output.info(chalk.yellow("No insights match the criteria"));
                return;
            }

            this.output(
                { insights, count: insights.length },
                () => {
                    output.info(chalk.bold(`\n🧠 Team Insights (${insights.length} found):`));
                    output.info("─".repeat(UI.TERMINAL_WIDTH));

                    insights.forEach((item, index) => {
                        const insight = item.value as TeamInsight;
                        const confidenceIcon = insight.confidence > TEAM_CONSTANTS.THROTTLE_THRESHOLD ? "🟢" : insight.confidence > TEAM_CONSTANTS.MEMORY_USAGE_WARNING ? "🟡" : "🔴";
                        
                        output.info(`\n${index + 1}. ${chalk.bold(insight.type)} ${confidenceIcon} (${(insight.confidence * 100).toFixed(0)}%)`);
                        output.info(`   Created: ${new Date(item.created_at).toLocaleString()}`);
                        output.info(`   Data: ${JSON.stringify(insight.data, null, 2).split("\n").map(line => "   " + line).join("\n").trim()}`);
                    });
                },
            );
        } catch (error) {
            this.handleError(error, "Failed to fetch insights");
        }
    }

    // Helper methods
    private validateTeamConfig(config: TeamValidationData): { valid: boolean; errors: string[] } {
        const errors: string[] = [];

        if (!config.__version) {
            errors.push("Missing __version field");
        }

        if (!config.deploymentType || !["development", "saas", "appliance"].includes(config.deploymentType)) {
            errors.push("Invalid deploymentType. Must be: development, saas, or appliance");
        }

        if (!config.goal || typeof config.goal !== "string") {
            errors.push("Missing or invalid goal");
        }

        if (!config.businessPrompt || typeof config.businessPrompt !== "string") {
            errors.push("Missing or invalid businessPrompt");
        }

        if (!config.resourceQuota || typeof config.resourceQuota !== "object") {
            errors.push("Missing or invalid resourceQuota");
        } else {
            if (typeof config.resourceQuota.gpuPercentage !== "number" || config.resourceQuota.gpuPercentage < 0 || config.resourceQuota.gpuPercentage > 100) {
                errors.push("Invalid gpuPercentage. Must be 0-100");
            }
            if (typeof config.resourceQuota.ramGB !== "number" || config.resourceQuota.ramGB <= 0) {
                errors.push("Invalid ramGB. Must be positive");
            }
        }

        if (!config.targetProfitPerMonth || typeof config.targetProfitPerMonth !== "string") {
            errors.push("Missing or invalid targetProfitPerMonth");
        } else {
            try {
                BigInt(config.targetProfitPerMonth);
            } catch {
                errors.push("targetProfitPerMonth must be a valid bigint string");
            }
        }

        if (!config.stats || typeof config.stats !== "object") {
            errors.push("Missing or invalid stats object");
        }

        return { valid: errors.length === 0, errors };
    }

    private displayTeamsTable(teams: Team[]): void {
        if (teams.length === 0) {
            output.info(chalk.yellow("No teams found"));
            return;
        }

        output.info(chalk.bold("\nTeams:"));
        output.info("");
        
        teams.forEach((team, index) => {
            const name = team.translations?.[0]?.name || "Unnamed Team";
            const config = team.config as TeamValidationData;
            const targetProfitUSD = Number(BigInt(config.targetProfitPerMonth)) / TEAM_CONSTANTS.MICROSECONDS_PER_SECOND;
            const activeInstances = config.stats?.activeInstances || 0;
            
            output.info(`${chalk.cyan((index + 1).toString().padStart(3))}. ${chalk.bold(name)}`);
            output.info(`     ID: ${chalk.gray(team.id)}`);
            output.info(`     Goal: ${config.goal}`);
            output.info(`     Type: ${config.deploymentType} | Target: $${targetProfitUSD}/mo | Active: ${activeInstances}`);
            output.info("");
        });
    }

    private createProgressBar(value: number, max: number, width = TEAM_CONSTANTS.DEFAULT_RESOURCE_LIMIT): string {
        const percentage = Math.min(100, Math.max(0, (value / max) * 100));
        const filled = Math.floor((percentage / 100) * width);
        const empty = width - filled;
        
        const filledChar = "█";
        const emptyChar = "░";
        
        const color = percentage >= TEAM_CONSTANTS.TABLE_WIDTH ? chalk.green : percentage >= TEAM_CONSTANTS.COLUMN_WIDTH ? chalk.yellow : chalk.red;
        
        return `[${color(filledChar.repeat(filled))}${chalk.gray(emptyChar.repeat(empty))}]`;
    }
}
