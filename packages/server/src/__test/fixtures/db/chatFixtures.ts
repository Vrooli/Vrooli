// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced as any with proper type cast for Prisma.chatUpsertArgs["create"]
import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { chatConfigFixtures } from "../../../../../shared/src/__test/fixtures/config/chatConfigFixtures.js";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Chat model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const chatDbIds = {
    chat1: generatePK(),
    chat2: generatePK(),
    chat3: generatePK(),
    invite1: generatePK(),
    invite2: generatePK(),
    message1: generatePK(),
    message2: generatePK(),
    participant1: generatePK(),
    participant2: generatePK(),
    participant3: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
};

/**
 * Minimal chat data for database creation
 */
export const minimalChatDb: Prisma.chatUpsertArgs["create"] = {
    id: chatDbIds.chat1,
    publicId: generatePublicId(),
    isPrivate: false,
    openToAnyoneWithInvite: true,
    config: chatConfigFixtures.minimal as any,
};

/**
 * Private chat with participants
 */
export const privateChatDb: Prisma.chatUpsertArgs["create"] = {
    id: chatDbIds.chat2,
    publicId: generatePublicId(),
    isPrivate: true,
    openToAnyoneWithInvite: false,
    config: chatConfigFixtures.variants.restrictedTeamSwarm as any,
    translations: {
        create: [
            {
                id: chatDbIds.translation1,
                language: "en",
                name: "Private Team Chat",
                description: "Discussion for team members only",
            },
        ],
    },
};

/**
 * Complete chat with all features
 */
export const completeChatDb: Prisma.chatUpsertArgs["create"] = {
    id: chatDbIds.chat3,
    publicId: generatePublicId(),
    isPrivate: false,
    openToAnyoneWithInvite: true,
    config: chatConfigFixtures.complete as any,
    translations: {
        create: [
            {
                id: chatDbIds.translation2,
                language: "en",
                name: "Project Discussion",
                description: "Open discussion about the project",
            },
            {
                id: chatDbIds.translation3,
                language: "es",
                name: "Discusión del Proyecto",
                description: "Discusión abierta sobre el proyecto",
            },
        ],
    },
};

/**
 * Enhanced test fixtures for Chat model following standard structure
 */
export const chatDbFixtures: DbTestFixtures<Prisma.chatUpsertArgs["create"]> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        isPrivate: false,
        openToAnyoneWithInvite: true,
        config: chatConfigFixtures.minimal as any,
    },
    complete: {
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
                    name: "Complete Chat Discussion",
                    description: "A comprehensive chat with full features",
                },
                {
                    id: generatePK(),
                    language: "es",
                    name: "Discusión Completa del Chat",
                    description: "Un chat integral con todas las funcionalidades",
                },
                {
                    id: generatePK(),
                    language: "fr",
                    name: "Discussion de Chat Complète",
                    description: "Un chat complet avec toutes les fonctionnalités",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required id and publicId
            isPrivate: false,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            isPrivate: "yes", // Should be boolean
            openToAnyoneWithInvite: "no", // Should be boolean
        },
        invalidTeamConnection: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            team: { connect: { id: "non-existent-team-id" } },
        },
        privateWithPublicInvite: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: true,
            openToAnyoneWithInvite: true, // Business logic violation
        },
    },
    edgeCases: {
        emptyPrivateChat: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfigFixtures.minimal as any,
        },
        chatWithManyTranslations: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.minimal as any,
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
        teamChatWithStructure: {
            id: generatePK(),
            publicId: generatePublicId(),
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
        },
        publicBotChat: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.variants.publicSwarm as any,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "AI Assistant Chat",
                    description: "Public chat with AI bot assistance",
                }],
            },
        },
        highLimitChat: {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.variants.highLimitSwarm as any,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "High-Performance Swarm Chat",
                    description: "Chat with elevated limits for complex operations",
                }],
            },
        },
    },
};

/**
 * Enhanced factory for creating chat database fixtures
 */
export class ChatDbFactory extends EnhancedDbFactory<Prisma.chatUpsertArgs["create"]> {
    
    /**
     * Get the test fixtures for Chat model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chatUpsertArgs["create"]> {
        return chatDbFixtures;
    }

    /**
     * Get Chat-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: chatDbIds.chat1, // Duplicate ID
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    teamId: BigInt("999999999999999"), // Non-existent team ID
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                },
            },
            validation: {
                requiredFieldMissing: chatDbFixtures.invalid.missingRequired,
                invalidDataType: chatDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "a".repeat(1000), // Name too long (max 128 chars)
                            description: "b".repeat(5000), // Description too long (max 2048 chars)
                        }],
                    },
                },
            },
            businessLogic: {
                privateWithPublicInvite: chatDbFixtures.invalid.privateWithPublicInvite,
            },
        };
    }

    /**
     * Generate fresh IDs for Chat model (no handle field)
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
        };
    }

    /**
     * Add team connection to a chat fixture
     */
    protected addTeamMemberships(data: Prisma.chatUpsertArgs["create"], teams: Array<{ teamId: string; role: string }>): Prisma.chatUpsertArgs["create"] {
        // For chats, we connect to the first team provided
        const team = teams[0];
        return {
            ...data,
            team: { connect: { id: BigInt(team.teamId) } },
        } as Prisma.chatUpsertArgs["create"];
    }

    /**
     * Chat-specific validation
     */
    protected validateSpecific(data: Prisma.chatUpsertArgs["create"]): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Chat
        if (!data.publicId) errors.push("Chat publicId is required");
        if (data.isPrivate === undefined) errors.push("Chat isPrivate flag is required");
        if (data.openToAnyoneWithInvite === undefined) errors.push("Chat openToAnyoneWithInvite flag is required");

        // Check business logic
        if (data.isPrivate && data.openToAnyoneWithInvite) {
            errors.push("Private chats should not be open to anyone with invite");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.chatUpsertArgs["create"]>): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal(overrides);
    }

    static createPrivate(overrides?: Partial<Prisma.chatUpsertArgs["create"]>): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal({
            isPrivate: true,
            openToAnyoneWithInvite: false,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Private Chat",
                    description: "Private discussion for selected members",
                }],
            },
            ...overrides,
        });
    }

    static createComplete(overrides?: Partial<Prisma.chatUpsertArgs["create"]>): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createComplete(overrides);
    }

    /**
     * Create chat with participants - requires user connections
     */
    static createWithParticipants(
        userIds: string[], 
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal({
            participants: {
                create: userIds.map((userId) => ({
                    id: generatePK(),
                    userId: BigInt(userId),
                })),
            },
            ...overrides,
        });
    }

    /**
     * Create chat with invites
     */
    static createWithInvites(
        creatorId: string,
        inviteMessages: string[] = ["Join our discussion!"],
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal({
            invites: {
                create: inviteMessages.map(message => ({
                    id: generatePK(),
                    message,
                    userId: BigInt(creatorId),
                })),
            },
            ...overrides,
        });
    }

    /**
     * Create chat with initial messages
     */
    static createWithMessages(
        messages: Array<{ userId: string; text: string; language?: string }>,
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal({
            messages: {
                create: messages.map(msg => ({
                    id: generatePK(),
                    config: {
                        __version: "1.0.0",
                        resources: [],
                    },
                    language: msg.language || "en",
                    text: msg.text,
                    userId: BigInt(msg.userId),
                })),
            },
            ...overrides,
        });
    }

    /**
     * Create chat with team connection
     */
    static createWithTeam(
        teamId: string,
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createWithRelationships({ 
            withTeams: [{ teamId, role: "Member" }], 
            overrides, 
        }).data;
    }

    /**
     * Create private DM chat between two users
     */
    static createPrivateDM(
        user1Id: string,
        user2Id: string,
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal({
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfigFixtures.minimal as any,
            participants: {
                create: [
                    {
                        id: generatePK(),
                        userId: BigInt(user1Id),
                    },
                    {
                        id: generatePK(),
                        userId: BigInt(user2Id),
                    },
                ],
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Private Message",
                    description: "Direct message conversation",
                }],
            },
            ...overrides,
        });
    }

    /**
     * Create team chat with specific configuration
     */
    static createTeamChat(
        teamId: string,
        config: {
            name: string;
            description?: string;
            isPrivate?: boolean;
            withHighLimits?: boolean;
        },
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        const chatConfig = config.withHighLimits 
            ? chatConfigFixtures.variants.highLimitSwarm
            : chatConfigFixtures.variants.restrictedTeamSwarm;

        return factory.createMinimal({
            teamId: BigInt(teamId),
            isPrivate: config.isPrivate ?? true,
            openToAnyoneWithInvite: false,
            config: chatConfig as any,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: config.name,
                    description: config.description || `Team chat for ${config.name}`,
                }],
            },
            ...overrides,
        });
    }

    /**
     * Create public chat with open invites
     */
    static createPublicChat(
        name: string,
        creatorId: string,
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        return factory.createMinimal({
            isPrivate: false,
            openToAnyoneWithInvite: true,
            config: chatConfigFixtures.variants.publicSwarm as any,
            participants: {
                create: [{
                    id: generatePK(),
                    userId: BigInt(creatorId),
                }],
            },
            invites: {
                create: [{
                    id: generatePK(),
                    message: `Join the ${name} discussion!`,
                    userId: BigInt(creatorId),
                }],
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name,
                    description: "Public discussion open to all",
                }],
            },
            ...overrides,
        });
    }

    /**
     * Create bot chat with AI configuration
     */
    static createBotChat(
        userId: string,
        botId: string,
        config?: {
            goal?: string;
            preferredModel?: string;
        },
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        const chatConfig = {
            ...chatConfigFixtures.complete,
            goal: config?.goal || "Assist the user with their questions",
            preferredModel: config?.preferredModel || "claude-3-opus",
            swarmLeader: botId,
        };

        return factory.createMinimal({
            isPrivate: true,
            openToAnyoneWithInvite: false,
            config: chatConfig as any,
            participants: {
                create: [
                    {
                        id: generatePK(),
                        userId: BigInt(userId),
                    },
                    {
                        id: generatePK(),
                        userId: BigInt(botId),
                    },
                ],
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "AI Assistant Chat",
                    description: "Personal conversation with AI assistant",
                }],
            },
            ...overrides,
        });
    }

    /**
     * Create chat with full relations
     */
    static createWithRelations(
        options: {
            userIds?: string[];
            teamId?: string;
            withMessages?: boolean;
            withInvites?: boolean;
        },
        overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
    ): Prisma.chatUpsertArgs["create"] {
        const factory = new ChatDbFactory();
        const baseData = factory.createMinimal(overrides);

        if (options.teamId) {
            baseData.teamId = BigInt(options.teamId);
        }

        if (options.userIds && options.userIds.length > 0) {
            baseData.participants = {
                create: options.userIds.map(userId => ({
                    id: generatePK(),
                    userId: BigInt(userId),
                })),
            };

            if (options.withMessages) {
                baseData.messages = {
                    create: [{
                        id: generatePK(),
                        config: { __version: "1.0.0", resources: [] },
                        language: "en",
                        text: "Welcome to the chat!",
                        userId: BigInt(options.userIds[0]),
                    }],
                };
            }

            if (options.withInvites) {
                baseData.invites = {
                    create: [{
                        id: generatePK(),
                        message: "You're invited to join our discussion",
                        userId: BigInt(options.userIds[0]),
                    }],
                };
            }
        }

        return baseData;
    }
}

/**
 * Helper to create chat with team connection
 */
export function createTeamChat(
    teamId: string,
    overrides?: Partial<Prisma.chatUpsertArgs["create"]>,
): Prisma.chatUpsertArgs["create"] {
    return {
        ...ChatDbFactory.createMinimal(overrides),
        teamId: BigInt(teamId),
    };
}

/**
 * Enhanced helper to seed multiple test chats with comprehensive options
 */
export async function seedTestChats(
    prisma: any,
    count = 3,
    options?: BulkSeedOptions & {
        isPrivate?: boolean;
        withMessages?: boolean;
        withInvites?: boolean;
        userIds?: string[];
    },
): Promise<BulkSeedResult<any>> {
    const factory = new ChatDbFactory();
    const chats = [];
    let withMessagesCount = 0;
    let withInvitesCount = 0;
    let teamCount = 0;

    for (let i = 0; i < count; i++) {
        const overrides = options?.overrides?.[i] || {};
        
        let chatData: Prisma.chatUpsertArgs["create"];
        
        if (options?.isPrivate) {
            chatData = factory.createMinimal({
                isPrivate: true,
                openToAnyoneWithInvite: false,
                ...overrides,
            });
        } else {
            chatData = factory.createMinimal(overrides);
        }

        // Add team connection if requested
        if (options?.teamId) {
            chatData.teamId = BigInt(options.teamId);
            teamCount++;
        }

        // Add participants if user IDs provided
        if (options?.userIds && options.userIds.length > 0) {
            chatData.participants = {
                create: options.userIds.map(userId => ({
                    id: generatePK(),
                    userId: BigInt(userId),
                })),
            };
        }

        // Add messages if requested
        if (options?.withMessages && options?.userIds && options.userIds.length > 0) {
            chatData.messages = {
                create: [{
                    id: generatePK(),
                    config: { __version: "1.0.0", resources: [] },
                    language: "en",
                    text: `Welcome to chat ${i + 1}! Let's start our discussion.`,
                    userId: BigInt(options.userIds[0]),
                }],
            };
            withMessagesCount++;
        }

        // Add invites if requested
        if (options?.withInvites && options?.userIds && options.userIds.length > 0) {
            chatData.invites = {
                create: [{
                    id: generatePK(),
                    message: `Join our chat ${i + 1} discussion`,
                    userId: BigInt(options.userIds[0]),
                }],
            };
            withInvitesCount++;
        }

        chats.push(await prisma.chat.create({ data: chatData }));
    }

    return {
        records: chats,
        summary: {
            total: chats.length,
            withAuth: 0, // Chats don't have auth directly
            bots: 0, // Chats don't have bots directly
            teams: teamCount,
        },
    };
}

/**
 * Helper to create a full chat setup for testing (backward compatibility)
 */
export async function seedTestChat(
    prisma: any,
    options: {
        userIds: string[];
        isPrivate?: boolean;
        withMessages?: boolean;
        withInvites?: boolean;
        teamId?: string;
    },
) {
    const chatData = ChatDbFactory.createWithRelations(
        {
            userIds: options.userIds,
            teamId: options.teamId,
            withMessages: options.withMessages,
            withInvites: options.withInvites,
        },
        {
            isPrivate: options.isPrivate ?? false,
        },
    );

    return await prisma.chat.create({ data: chatData });
}
