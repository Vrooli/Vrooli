import { type ApiClient } from "../../utils/client.js";
import { type ConfigManager } from "../../utils/config.js";
import type { CompletionProvider, CompletionResult, CompletionContext } from "../types.js";
import type { 
    ResourceSearchResult, 
    UserSearchResult,
    TeamSearchResult,
    ChatSearchResult,
} from "@vrooli/shared";

export class DynamicProvider implements CompletionProvider {
    constructor(
        private client: ApiClient,
        private config: ConfigManager,
    ) {}
    
    canHandle(context: CompletionContext): boolean {
        return context.type === "resource";
    }
    
    async getCompletions(context: CompletionContext): Promise<CompletionResult[]> {
        if (!context.resourceType) return [];
        
        try {
            switch (context.resourceType) {
                case "routine":
                    return this.fetchRoutines(context.partial);
                case "agent":
                    return this.fetchAgents(context.partial);
                case "team":
                    return this.fetchTeams(context.partial);
                case "chat":
                    return this.fetchChats(context.partial);
                case "bot":
                    return this.fetchBots(context.partial);
                case "history":
                    return this.fetchHistory(context.partial);
                default:
                    return [];
            }
        } catch (error) {
            // Fail silently for completions - we don't want to break the shell
            if (this.config.isDebug()) {
                console.error("Completion fetch error:", error);
            }
            return [];
        }
    }
    
    private async fetchRoutines(partial: string): Promise<CompletionResult[]> {
        try {
            const response = await this.client.post<ResourceSearchResult>("/api/resources", {
                take: 20,
                searchString: partial,
                latestVersionResourceSubType: ["RoutineInformational", "RoutineGenerate", "RoutineApi", "RoutineCode"],
            });
            
            if (!response.edges) return [];
            
            return response.edges.map(edge => ({
                value: edge.node.publicId || edge.node.id,
                description: edge.node.translatedName || "Unnamed routine",
                type: "resource" as const,
                metadata: { 
                    resourceType: "routine",
                    id: edge.node.id,
                },
            }));
        } catch (error) {
            return [];
        }
    }
    
    private async fetchAgents(partial: string): Promise<CompletionResult[]> {
        try {
            const response = await this.client.post<ResourceSearchResult>("/api/resources", {
                take: 20,
                searchString: partial,
                latestVersionResourceSubType: "AgentSpec",
            });
            
            if (!response.edges) return [];
            
            return response.edges.map(edge => {
                const config = edge.node.versions?.[0]?.config as unknown as Record<string, unknown>;
                const agentSpec = config?.agentSpec as Record<string, unknown> | undefined;
                const role = (agentSpec?.role as string) || "unknown";
                return {
                    value: edge.node.publicId || edge.node.id,
                    description: `${edge.node.translatedName || "Unnamed agent"} (${role})`,
                    type: "resource" as const,
                    metadata: { 
                        resourceType: "agent",
                        id: edge.node.id,
                        role,
                    },
                };
            });
        } catch (error) {
            return [];
        }
    }
    
    private async fetchTeams(_partial: string): Promise<CompletionResult[]> {
        try {
            const response = await this.client.post<TeamSearchResult>("/teams", {
                take: 20,
                searchString: _partial,
            });
            
            if (!response.edges) return [];
            
            return response.edges.map(edge => {
                const config = edge.node.config as unknown as Record<string, unknown>;
                const deploymentType = (config?.deploymentType as string) || "unknown";
                return {
                    value: edge.node.id,
                    description: `${edge.node.translations?.[0]?.name || "Unnamed team"} (${deploymentType})`,
                    type: "resource" as const,
                    metadata: { 
                        resourceType: "team",
                        id: edge.node.id,
                        deploymentType,
                    },
                };
            });
        } catch (error) {
            return [];
        }
    }
    
    private async fetchChats(partial: string): Promise<CompletionResult[]> {
        try {
            const response = await this.client.post<ChatSearchResult>("/chats", {
                take: 20,
                searchString: partial,
                visibility: "Own",
            });
            
            if (!response.edges) return [];
            
            return response.edges.map(edge => ({
                value: edge.node.id,
                description: edge.node.translations?.[0]?.name || "Unnamed chat",
                type: "resource" as const,
                metadata: { 
                    resourceType: "chat",
                    id: edge.node.id,
                    participantsCount: edge.node.participantsCount,
                },
            }));
        } catch (error) {
            return [];
        }
    }
    
    private async fetchBots(partial: string): Promise<CompletionResult[]> {
        try {
            const response = await this.client.post<UserSearchResult>("/users", {
                take: 20,
                searchString: partial,
                isBot: true,
            });
            
            if (!response.edges) return [];
            
            return response.edges.map(edge => ({
                value: edge.node.id,
                description: edge.node.name || edge.node.handle || "Unnamed bot",
                type: "resource" as const,
                metadata: { 
                    resourceType: "bot",
                    id: edge.node.id,
                    handle: edge.node.handle,
                },
            }));
        } catch (error) {
            return [];
        }
    }
    
    private async fetchHistory(_partial: string): Promise<CompletionResult[]> {
        // For history entries, we'll need to implement this once the history manager is ready
        // For now, return empty array
        return [];
    }
    
    /**
     * Pre-fetch common resources to improve completion speed
     */
    async prefetch(): Promise<void> {
        // This could be called on first completion to cache common resources
        try {
            await Promise.all([
                this.fetchRoutines(""),
                this.fetchBots(""),
                this.fetchTeams(""),
            ]);
        } catch (error) {
            // Ignore prefetch errors
        }
    }
}
