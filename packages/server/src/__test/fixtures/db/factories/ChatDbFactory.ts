import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type chat, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
// TODO: Update this import path when chatConfigFixtures is available
const chatConfigFixtures = {
    minimal: { __version: "1.0.0" },
    complete: { __version: "1.0.0", goal: "Default chat goal" },
    variants: {
        restrictedTeamSwarm: { __version: "1.0.0", isRestricted: true },
        publicSwarm: { __version: "1.0.0", isPublic: true },
        highLimitSwarm: { __version: "1.0.0", limits: { maxToolCalls: 1000 } },
    },
};

interface ChatRelationConfig extends RelationConfig {
    withTeam?: boolean;
    teamId?: string | bigint;
    withParticipants?: boolean | number;
    participantIds?: (string | bigint)[];
    withMessages?: boolean | number;
    withInvites?: boolean | number;
    translations?: Array<{ language: string; name: string; description?: string }>;
}

/**
 * Database fixture factory for Chat model
 * Handles team chats, private chats, and AI bot integration with comprehensive relationship support
 */
export class ChatDbFactory extends DatabaseFixtureFactory<
    chat,
    Prisma.chatCreateInput,
    Prisma.chatInclude,
    Prisma.chatUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("chat", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.chat;
    }

    protected getMinimalData(overrides?: Partial<Prisma.chatCreateInput>): Prisma.chatCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.minimal as any,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.chatCreateInput>): Prisma.chatCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.complete as any,
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Project Discussion",
                        description: "Open discussion about the project",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Discusión del Proyecto",
                        description: "Discusión abierta sobre el proyecto",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Create a private chat with specific configuration
     */
    async createPrivate(overrides?: Partial<Prisma.chatCreateInput>): Promise<chat> {
        const data: Prisma.chatCreateInput = {
            ...this.getMinimalData(),
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfigFixtures.variants.restrictedTeamSwarm as any,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Private Team Chat",
                    description: "Discussion for team members only",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.chat.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create a team chat with enhanced configuration
     */
    async createTeamChat(teamId: string, overrides?: Partial<Prisma.chatCreateInput>): Promise<chat> {
        const data: Prisma.chatCreateInput = {
            ...this.getMinimalData(),
            team: { connect: { id: BigInt(teamId) } },
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfigFixtures.variants.restrictedTeamSwarm as any,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Team Strategy Discussion",
                    description: "Private team chat for strategic planning and coordination",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.chat.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create a bot chat for AI assistance
     */
    async createBotChat(userId: string, botId: string, overrides?: Partial<Prisma.chatCreateInput>): Promise<chat> {
        const data: Prisma.chatCreateInput = {
            ...this.getMinimalData(),
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: {
                ...chatConfigFixtures.complete,
                goal: "Assist the user with their questions",
                preferredModel: "claude-3-opus",
                swarmLeader: botId,
            } as any,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "AI Assistant Chat",
                    description: "Personal conversation with AI assistant",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.chat.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    protected getDefaultInclude(): Prisma.chatInclude {
        return {
            translations: true,
            participants: {
                select: {
                    id: true,
                    userId: true,
                    hasUnread: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                            handle: true,
                            isBot: true,
                        },
                    },
                },
            },
            messages: {
                take: 5,
                orderBy: { createdAt: "desc" },
                select: {
                    id: true,
                    text: true,
                    createdAt: true,
                    user: {
                        select: {
                            id: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            invites: {
                where: { status: "Pending" },
                select: {
                    id: true,
                    message: true,
                    status: true,
                    user: {
                        select: {
                            id: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            team: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                    translations: {
                        where: { language: "en" },
                        select: {
                            name: true,
                        },
                        take: 1,
                    },
                },
            },
            _count: {
                select: {
                    participants: true,
                    messages: true,
                    invites: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.chatCreateInput,
        config: ChatRelationConfig,
        tx: any,
    ): Promise<Prisma.chatCreateInput> {
        const data = { ...baseData };

        // Handle team relationship
        if (config.withTeam && config.teamId) {
            data.team = { connect: { id: BigInt(config.teamId) } };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        // Handle participants
        if (config.withParticipants && config.participantIds && config.participantIds.length > 0) {
            const participantCount = typeof config.withParticipants === "number" 
                ? Math.min(config.withParticipants, config.participantIds.length)
                : config.participantIds.length;
                
            data.participants = {
                create: config.participantIds.slice(0, participantCount).map(userId => ({
                    id: generatePK(),
                    userId: BigInt(userId),
                    hasUnread: false,
                })),
            };
        }

        // Handle initial messages
        if (config.withMessages && config.participantIds && config.participantIds.length > 0) {
            const messageCount = typeof config.withMessages === "number" ? config.withMessages : 1;
            const senderId = config.participantIds[0];
            
            data.messages = {
                create: Array.from({ length: messageCount }, (_, i) => ({
                    id: generatePK(),
                    versionIndex: i,
                    language: "en",
                    text: `Welcome to the chat! This is message ${i + 1}.`,
                    userId: BigInt(senderId),
                    config: {
                        __version: "1.0.0",
                        role: "user",
                        turnId: i + 1,
                    },
                })),
            };
        }

        // Handle invites
        if (config.withInvites && config.participantIds && config.participantIds.length > 0) {
            const inviteCount = typeof config.withInvites === "number" ? config.withInvites : 1;
            const creatorId = config.participantIds[0];
            
            data.invites = {
                create: Array.from({ length: inviteCount }, (_, i) => ({
                    id: generatePK(),
                    message: `Join our discussion! Invite ${i + 1}`,
                    userId: BigInt(creatorId),
                    status: "Pending",
                })),
            };
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createPrivateDM(user1Id: string | bigint, user2Id: string | bigint): Promise<chat> {
        return this.createWithRelations({
            overrides: {
                isPrivate: true,
                openToAnyoneWithInvite: false,
                config: chatConfigFixtures.minimal as any,
            },
            withParticipants: true,
            participantIds: [user1Id, user2Id],
            translations: [{
                language: "en",
                name: "Private Message",
                description: "Direct message conversation",
            }],
        });
    }

    async createPublicChat(name: string, creatorId: string | bigint): Promise<chat> {
        return this.createWithRelations({
            overrides: {
                isPrivate: false,
                openToAnyoneWithInvite: true,
                config: chatConfigFixtures.variants.publicSwarm as any,
            },
            withParticipants: true,
            participantIds: [creatorId],
            withMessages: 1,
            withInvites: 1,
            translations: [{
                language: "en",
                name,
                description: "Public discussion open to all",
            }],
        });
    }

    async createHighPerformanceChat(teamId: string | bigint): Promise<chat> {
        return this.createWithRelations({
            overrides: {
                team: { connect: { id: BigInt(teamId) } },
                isPrivate: true,
                openToAnyoneWithInvite: false,
                config: chatConfigFixtures.variants.highLimitSwarm as any,
            },
            withTeam: true,
            teamId,
            translations: [{
                language: "en",
                name: "High-Performance Swarm Chat",
                description: "Chat with elevated limits for complex operations",
            }],
        });
    }

    protected async checkModelConstraints(record: chat): Promise<string[]> {
        const violations: string[] = [];
        
        // Check publicId uniqueness
        if (record.publicId) {
            const duplicate = await this.prisma.chat.findFirst({
                where: { 
                    publicId: record.publicId,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push("PublicId must be unique");
            }
        }

        // Check business logic: private chats should not be open to anyone with invite
        if (record.isPrivate && record.openToAnyoneWithInvite) {
            violations.push("Private chats should not be open to anyone with invite");
        }

        // Check config structure
        if (!record.config || typeof record.config !== "object") {
            violations.push("Chat config is required and must be an object");
        }

        // Check team relationship exists if teamId is set
        if (record.teamId) {
            const team = await this.prisma.team.findUnique({
                where: { id: record.teamId },
            });
            if (!team) {
                violations.push("Referenced team does not exist");
            }
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId
                isPrivate: false,
                openToAnyoneWithInvite: true,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                isPrivate: "yes", // Should be boolean
                openToAnyoneWithInvite: "no", // Should be boolean
                config: "not an object", // Should be object
            },
            duplicatePublicId: {
                id: generatePK(),
                publicId: "existing_public_id", // Assumes this exists
                isPrivate: false,
                openToAnyoneWithInvite: true,
                config: chatConfigFixtures.minimal as any,
            },
            privateWithPublicInvite: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: true,
                openToAnyoneWithInvite: true, // Business logic violation
                config: chatConfigFixtures.minimal as any,
            },
            invalidTeamReference: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: true,
                openToAnyoneWithInvite: false,
                teamId: BigInt("999999999999999"), // Non-existent team
                config: chatConfigFixtures.minimal as any,
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.chatCreateInput> {
        return {
            emptyPrivateChat: {
                ...this.getMinimalData(),
                isPrivate: true,
                openToAnyoneWithInvite: false,
            },
            chatWithManyTranslations: {
                ...this.getMinimalData(),
                translations: {
                    create: [
                        { id: generatePK(), language: "en", name: "Multi Lang Chat", description: "English description" },
                        { id: generatePK(), language: "es", name: "Chat Multi Idioma", description: "Descripción en español" },
                        { id: generatePK(), language: "fr", name: "Chat Multi Langue", description: "Description française" },
                        { id: generatePK(), language: "de", name: "Mehrsprachiger Chat", description: "Deutsche Beschreibung" },
                        { id: generatePK(), language: "ja", name: "多言語チャット", description: "日本語の説明" },
                        { id: generatePK(), language: "zh", name: "多语言聊天", description: "中文描述" },
                    ],
                },
            },
            complexBotConfiguration: {
                ...this.getMinimalData(),
                isPrivate: true,
                openToAnyoneWithInvite: false,
                config: {
                    ...chatConfigFixtures.complete,
                    goal: "Complex AI assistance with multiple tool integrations",
                    preferredModel: "claude-3-opus",
                    limits: {
                        maxToolCallsPerBotResponse: 25,
                        maxToolCalls: 500,
                        maxCreditsPerBotResponse: "500",
                        maxCredits: "10000",
                    },
                } as any,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "AI Complex Assistant",
                        description: "Advanced AI chat with complex tool integration",
                    }],
                },
            },
            unicodeContent: {
                ...this.getMinimalData(),
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Unicode Test Chat 🚀🤖💬",
                        description: "Testing with emoji and special characters: ñáéíóú çñü",
                    }],
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            participants: true,
            messages: true,
            invites: true,
        };
    }

    protected async deleteRelatedRecords(
        record: chat & {
            messages?: any[];
            invites?: any[];
            participants?: any[];
            translations?: any[];
        },
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete messages
        if (record.messages?.length) {
            await tx.chat_message.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete invites
        if (record.invites?.length) {
            await tx.chat_invite.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete participants
        if (record.participants?.length) {
            await tx.chat_participants.deleteMany({
                where: { chatId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.chat_translation.deleteMany({
                where: { chatId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createChatDbFactory = (prisma: PrismaClient) => new ChatDbFactory(prisma);
