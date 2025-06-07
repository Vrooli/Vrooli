import { type PrismaClient } from "@prisma/client";
import { type Logger } from "winston";
import { type Routine, type RoutineMetadata, generatePk } from "@vrooli/shared";

/**
 * Routine storage service that integrates with Vrooli's database
 * 
 * This service provides access to stored routines/workflows from the database,
 * converting them into the format expected by the execution architecture.
 */
export class RoutineStorageService {
    private readonly prisma: PrismaClient;
    private readonly logger: Logger;

    constructor(prisma: PrismaClient, logger: Logger) {
        this.prisma = prisma;
        this.logger = logger;
    }

    /**
     * Loads a routine by ID from the database
     */
    async loadRoutine(routineId: string): Promise<Routine> {
        this.logger.debug("[RoutineStorageService] Loading routine", { routineId });

        try {
            // Find the resource by ID (could be resource ID or version ID)
            const resourceVersion = await this.prisma.resource_version.findFirst({
                where: {
                    OR: [
                        { id: BigInt(routineId) },
                        { publicId: routineId },
                        { root: { publicId: routineId } },
                    ],
                },
                include: {
                    root: true,
                    translations: {
                        include: {
                            language: true,
                        },
                    },
                    resourceList: {
                        include: {
                            translations: true,
                        },
                    },
                },
                orderBy: {
                    versionIndex: "desc", // Get latest version
                },
            });

            if (!resourceVersion) {
                throw new Error(`Routine not found: ${routineId}`);
            }

            // Convert to execution format
            const routine: Routine = {
                id: resourceVersion.publicId,
                type: this.inferRoutineType(resourceVersion),
                version: resourceVersion.versionLabel,
                name: this.extractName(resourceVersion),
                definition: this.parseDefinition(resourceVersion),
                metadata: this.buildMetadata(resourceVersion),
            };

            this.logger.info("[RoutineStorageService] Routine loaded", {
                routineId,
                name: routine.name,
                type: routine.type,
            });

            return routine;

        } catch (error) {
            this.logger.error("[RoutineStorageService] Failed to load routine", {
                routineId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Searches for routines by criteria
     */
    async searchRoutines(criteria: {
        query?: string;
        userId?: string;
        teamId?: string;
        tags?: string[];
        limit?: number;
        offset?: number;
    }): Promise<Routine[]> {
        this.logger.debug("[RoutineStorageService] Searching routines", criteria);

        try {
            const { query, userId, teamId, tags, limit = 20, offset = 0 } = criteria;

            const resourceVersions = await this.prisma.resource_version.findMany({
                where: {
                    AND: [
                        // Only routines/workflows
                        {
                            root: {
                                resourceType: {
                                    in: ["Routine", "Standard"],
                                },
                            },
                        },
                        // Search query
                        query ? {
                            OR: [
                                {
                                    translations: {
                                        some: {
                                            name: {
                                                contains: query,
                                                mode: "insensitive",
                                            },
                                        },
                                    },
                                },
                                {
                                    translations: {
                                        some: {
                                            description: {
                                                contains: query,
                                                mode: "insensitive",
                                            },
                                        },
                                    },
                                },
                            ],
                        } : {},
                        // User filter
                        userId ? {
                            root: {
                                ownedByUserId: BigInt(userId),
                            },
                        } : {},
                        // Team filter
                        teamId ? {
                            root: {
                                ownedByTeamId: BigInt(teamId),
                            },
                        } : {},
                        // Tags filter
                        tags && tags.length > 0 ? {
                            tags: {
                                some: {
                                    tag: {
                                        tag: {
                                            in: tags,
                                        },
                                    },
                                },
                            },
                        } : {},
                        // Only public or owned resources
                        {
                            root: {
                                isPrivate: false,
                            },
                        },
                    ],
                },
                include: {
                    root: true,
                    translations: {
                        include: {
                            language: true,
                        },
                    },
                },
                orderBy: {
                    root: {
                        score: "desc",
                    },
                },
                take: limit,
                skip: offset,
            });

            const routines = resourceVersions.map(rv => ({
                id: rv.publicId,
                type: this.inferRoutineType(rv),
                version: rv.versionLabel,
                name: this.extractName(rv),
                definition: this.parseDefinition(rv),
                metadata: this.buildMetadata(rv),
            }));

            this.logger.info("[RoutineStorageService] Search completed", {
                criteria,
                resultCount: routines.length,
            });

            return routines;

        } catch (error) {
            this.logger.error("[RoutineStorageService] Search failed", {
                criteria,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Checks if user has permission to execute a routine
     */
    async checkExecutePermission(routineId: string, userId: string): Promise<boolean> {
        try {
            const resourceVersion = await this.prisma.resource_version.findFirst({
                where: {
                    OR: [
                        { publicId: routineId },
                        { root: { publicId: routineId } },
                    ],
                },
                include: {
                    root: true,
                },
            });

            if (!resourceVersion) {
                return false;
            }

            // Check permissions
            const resource = resourceVersion.root;
            
            // Owner can always execute
            if (resource.ownedByUserId === BigInt(userId)) {
                return true;
            }

            // Check if user is part of owning team
            if (resource.ownedByTeamId) {
                const teamMember = await this.prisma.team_member.findFirst({
                    where: {
                        teamId: resource.ownedByTeamId,
                        userId: BigInt(userId),
                    },
                });
                if (teamMember) {
                    return true;
                }
            }

            // Check if resource is public
            if (!resource.isPrivate) {
                return true;
            }

            return false;

        } catch (error) {
            this.logger.error("[RoutineStorageService] Permission check failed", {
                routineId,
                userId,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Private helper methods
     */
    private inferRoutineType(resourceVersion: any): string {
        // Look at the resource definition to determine type
        const definition = this.parseDefinition(resourceVersion);
        
        if (definition.nodes && definition.edges) {
            return "native"; // Vrooli native format
        }
        
        if (definition.process && definition.startEvent) {
            return "bpmn";
        }
        
        if (definition.chain || definition.tools) {
            return "langchain";
        }
        
        if (definition.workflows) {
            return "temporal";
        }
        
        return "native"; // Default
    }

    private extractName(resourceVersion: any): string {
        // Get name from translations (prefer English)
        const englishTranslation = resourceVersion.translations?.find(
            (t: any) => t.language.code === "en"
        );
        
        if (englishTranslation?.name) {
            return englishTranslation.name;
        }
        
        // Fallback to any translation
        const firstTranslation = resourceVersion.translations?.[0];
        if (firstTranslation?.name) {
            return firstTranslation.name;
        }
        
        return `Routine ${resourceVersion.publicId}`;
    }

    private parseDefinition(resourceVersion: any): Record<string, unknown> {
        try {
            // The definition should be in the JSON data field
            if (typeof resourceVersion.data === "string") {
                return JSON.parse(resourceVersion.data);
            }
            
            if (typeof resourceVersion.data === "object") {
                return resourceVersion.data as Record<string, unknown>;
            }
            
            return {};
        } catch (error) {
            this.logger.warn("[RoutineStorageService] Failed to parse definition", {
                resourceVersionId: resourceVersion.id,
                error: error instanceof Error ? error.message : String(error),
            });
            return {};
        }
    }

    private buildMetadata(resourceVersion: any): RoutineMetadata {
        const resource = resourceVersion.root;
        
        return {
            author: resource.createdById ? resource.createdById.toString() : "unknown",
            createdAt: resource.createdAt,
            updatedAt: resource.updatedAt,
            tags: [], // Would need to join tags if needed
            complexity: this.estimateComplexity(this.parseDefinition(resourceVersion)),
            version: resourceVersion.versionLabel,
            description: this.extractDescription(resourceVersion),
        };
    }

    private extractDescription(resourceVersion: any): string {
        const englishTranslation = resourceVersion.translations?.find(
            (t: any) => t.language.code === "en"
        );
        
        return englishTranslation?.description || "";
    }

    private estimateComplexity(definition: Record<string, unknown>): "simple" | "moderate" | "complex" {
        const nodeCount = Array.isArray(definition.nodes) ? definition.nodes.length : 0;
        const edgeCount = Array.isArray(definition.edges) ? definition.edges.length : 0;
        
        const totalElements = nodeCount + edgeCount;
        
        if (totalElements <= 5) return "simple";
        if (totalElements <= 15) return "moderate";
        return "complex";
    }
}