/**
 * Agent Database Factory
 * 
 * Creates agent records in the test database as bot users
 */

import { type Prisma } from "@prisma/client";
import { BotConfig, generatePublicId, ModelStrategy, type BotConfigObject, type EmitAction, type InvokeAction, type RoutineAction } from "@vrooli/shared";
import { DbProvider } from "../../../../db/provider.js";
import { logger } from "../../../../events/logger.js";
import type { AgentTestData } from "../../types.js";

export class AgentDbFactory {
    async create(agentData: AgentTestData): Promise<any> {
        // Create BotConfigObject from agent test data
        const botConfig: BotConfigObject = {
            __version: "1.0",
            resources: [],
            modelConfig: agentData.resources?.preferredModel ? {
                strategy: ModelStrategy.FIXED,
                preferredModel: agentData.resources.preferredModel,
                offlineOnly: false,
            } : undefined,
            maxTokens: undefined, // Can be added to AgentTestData if needed
            agentSpec: agentData.behaviors && agentData.behaviors.length > 0 ? {
                goal: agentData.goal,
                behaviors: agentData.behaviors.map(b => ({
                    trigger: {
                        topic: b.trigger.topic,
                        when: JSON.stringify(b.trigger.conditions),
                    },
                    action: ((): RoutineAction | InvokeAction | EmitAction => {
                        // Map invalid test action types to valid BehaviourSpec action types
                        const actionType = b.action.type;
                        if (actionType === "decision" || actionType === "accumulate") {
                            // Map to InvokeAction
                            return {
                                type: "invoke",
                                purpose: b.action.label || `Agent ${actionType} reasoning`,
                            };
                        } else if (actionType === "routine") {
                            // RoutineAction
                            return {
                                type: "routine",
                                routineId: b.action.id,
                                label: b.action.label || "Routine action",
                            };
                        } else if (actionType === "emit") {
                            // EmitAction
                            return {
                                type: "emit",
                                eventType: b.action.topic || "default.event",
                            };
                        } else {
                            // Fallback to invoke for unknown types
                            return {
                                type: "invoke",
                                purpose: b.action.label || "Agent reasoning",
                            };
                        }
                    })(),
                })),
                subscriptions: agentData.subscriptions, // Already an array of strings
            } : undefined,
        };

        // Create agent as a bot user record
        const user = await DbProvider.get().user.create({
            data: {
                id: BigInt(agentData.id),
                publicId: generatePublicId(),
                name: agentData.name,
                isBot: true,
                botSettings: botConfig as unknown as Prisma.InputJsonValue,
                isPrivate: false,
                languages: ["en"],
                createdAt: agentData.created_at,
                updatedAt: agentData.updated_at,
            },
        });

        return {
            ...user,
            __typename: "Agent",
            // Transform back to test-friendly format
            goal: agentData.goal,
            subscriptions: agentData.subscriptions,
            behaviors: agentData.behaviors,
            resources: agentData.resources,
            teamId: agentData.teamId,
        };
    }

    async createBatch(agents: any[]): Promise<any[]> {
        return Promise.all(agents.map(a => this.create(a)));
    }

    async update(id: string, data: Partial<AgentTestData>): Promise<any> {
        // Update the bot user record
        const updateData: any = {};

        if (data.name) {
            updateData.name = data.name;
        }

        // If updating behaviors, resources, etc., rebuild botSettings
        if (data.behaviors || data.resources || data.goal || data.subscriptions) {
            const currentUser = await DbProvider.get().user.findUnique({
                where: { id: BigInt(id) },
                select: { botSettings: true },
            });

            const currentBotConfig = BotConfig.parse({ botSettings: currentUser?.botSettings as unknown as BotConfigObject }, logger);

            const updatedBotConfig: BotConfigObject = {
                ...currentBotConfig.export(),
                modelConfig: data.resources?.preferredModel ? {
                    strategy: ModelStrategy.FIXED,
                    preferredModel: data.resources.preferredModel,
                    offlineOnly: false,
                } : currentBotConfig.modelConfig,
                agentSpec: {
                    goal: data.goal || currentBotConfig.agentSpec?.goal || "",
                    behaviors: data.behaviors ? data.behaviors.map(b => ({
                        trigger: {
                            topic: b.trigger.topic,
                            when: JSON.stringify(b.trigger.conditions),
                        },
                        action: ((): RoutineAction | InvokeAction | EmitAction => {
                            // Map invalid test action types to valid BehaviourSpec action types
                            const actionType = b.action.type;
                            if (actionType === "decision" || actionType === "accumulate") {
                                // Map to InvokeAction
                                return {
                                    type: "invoke",
                                    purpose: b.action.label || `Agent ${actionType} reasoning`,
                                };
                            } else if (actionType === "routine") {
                                // RoutineAction
                                return {
                                    type: "routine",
                                    routineId: b.action.id,
                                    label: b.action.label || "Routine action",
                                };
                            } else if (actionType === "emit") {
                                // EmitAction
                                return {
                                    type: "emit",
                                    eventType: b.action.topic || "default.event",
                                };
                            } else {
                                // Fallback to invoke for unknown types
                                return {
                                    type: "invoke",
                                    purpose: b.action.label || "Agent reasoning",
                                };
                            }
                        })(),
                    })) : currentBotConfig.agentSpec?.behaviors || [],
                    subscriptions: data.subscriptions || currentBotConfig.agentSpec?.subscriptions || [],
                },
            };

            updateData.botSettings = updatedBotConfig;
        }

        return DbProvider.get().user.update({
            where: { id: BigInt(id) },
            data: updateData,
        });
    }

    async delete(id: string): Promise<void> {
        await DbProvider.get().user.delete({
            where: { id: BigInt(id) },
        });
    }
}
