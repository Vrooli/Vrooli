import { generatePK, generatePublicId, CommentFor } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Comment model - used for seeding test data
 * These follow Prisma's shape for database operations
 * 
 * Comments support polymorphic relationships through the commentFor field
 * and can be attached to: Issue, PullRequest, ResourceVersion
 */

// Consistent IDs for testing
export const commentDbIds = {
    comment1: generatePK(),
    comment2: generatePK(),
    comment3: generatePK(),
    comment4: generatePK(),
    comment5: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
    translation4: generatePK(),
    user1: generatePK(),
    user2: generatePK(),
    team1: generatePK(),
    issue1: generatePK(),
    resourceVersion1: generatePK(),
    pullRequest1: generatePK(),
};

/**
 * Enhanced test fixtures for Comment model following standard structure
 */
export const commentDbFixtures: DbTestFixtures<Prisma.commentUncheckedCreateInput> = {
    minimal: {
        id: generatePK(),
        score: 0,
        bookmarks: 0,
        translations: {
            create: [{
                id: generatePK(),
                language: "en",
                text: "Test comment",
            }],
        },
    },
    complete: {
        id: generatePK(),
        score: 5,
        bookmarks: 3,
        ownedByUserId: commentDbIds.user1,
        issueId: commentDbIds.issue1,
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    text: "This is a comprehensive test comment with detailed content.",
                },
                {
                    id: generatePK(),
                    language: "es",
                    text: "Este es un comentario de prueba completo con contenido detallado.",
                },
            ],
        },
        reactions: {
            create: [{
                id: generatePK(),
                emoji: "👍",
                byId: commentDbIds.user2,
            }],
        },
        bookmarkedBy: {
            create: [{
                id: generatePK(),
                userId: commentDbIds.user2,
            }],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required translations
            score: 0,
            bookmarks: 0,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            score: "not-a-number", // Should be number
            bookmarks: "not-a-number", // Should be number
            ownedByUserId: "invalid-id", // Should be valid BigInt
            translations: "not-an-object", // Should be relation
        },
        emptyTranslations: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [], // Empty translations array
            },
        },
        invalidTranslationText: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "a".repeat(32769), // Exceeds VARCHAR(32768) limit
                }],
            },
        },
        invalidLanguageCode: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "invalid", // Exceeds VARCHAR(3) limit
                    text: "Test comment",
                }],
            },
        },
        bothOwnerTypes: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            ownedByUserId: commentDbIds.user1,
            ownedByTeamId: commentDbIds.team1, // Should not have both
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Test comment",
                }],
            },
        },
    },
    edgeCases: {
        maxLengthText: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "a".repeat(32768), // Maximum text length
                }],
            },
        },
        negativeScore: {
            id: generatePK(),
            score: -100,
            bookmarks: 0,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Heavily downvoted comment",
                }],
            },
        },
        highScore: {
            id: generatePK(),
            score: 999999,
            bookmarks: 50000,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Viral comment with extremely high engagement",
                }],
            },
        },
        multipleParentLevels: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            parentId: commentDbIds.comment1,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Reply to a reply (nested comment)",
                }],
            },
        },
        multiLanguageTranslations: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            translations: {
                create: [
                    { id: generatePK(), language: "en", text: "English comment" },
                    { id: generatePK(), language: "es", text: "Comentario en español" },
                    { id: generatePK(), language: "fr", text: "Commentaire en français" },
                    { id: generatePK(), language: "de", text: "Kommentar auf Deutsch" },
                    { id: generatePK(), language: "ja", text: "日本語のコメント" },
                ],
            },
        },
        teamOwnedComment: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            ownedByTeamId: commentDbIds.team1,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Team-authored comment",
                }],
            },
        },
        orphanedComment: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            // No parent relationships (orphaned)
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Orphaned comment with no parent object",
                }],
            },
        },
        commentOnMultipleObjects: {
            id: generatePK(),
            score: 0,
            bookmarks: 0,
            issueId: commentDbIds.issue1,
            resourceVersionId: commentDbIds.resourceVersion1,
            pullRequestId: commentDbIds.pullRequest1,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Comment linked to multiple objects (edge case)",
                }],
            },
        },
    },
};

/**
 * Enhanced factory for creating comment database fixtures
 */
export class CommentDbFactory extends EnhancedDbFactory<Prisma.commentUncheckedCreateInput> {
    
    /**
     * Get the test fixtures for Comment model
     */
    protected getFixtures(): DbTestFixtures<Prisma.commentUncheckedCreateInput> {
        return commentDbFixtures;
    }

    /**
     * Get Comment-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: commentDbIds.comment1, // Duplicate ID
                    score: 0,
                    bookmarks: 0,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Duplicate ID comment",
                        }],
                    },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    score: 0,
                    bookmarks: 0,
                    ownedByUserId: BigInt("9999999999999999"), // Non-existent ID
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Foreign key violation comment",
                        }],
                    },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    score: 0,
                    bookmarks: 0,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "", // Empty language violates constraint
                            text: "Check constraint violation",
                        }],
                    },
                },
            },
            validation: {
                requiredFieldMissing: commentDbFixtures.invalid.missingRequired,
                invalidDataType: commentDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    score: Number.MAX_SAFE_INTEGER + 1, // Out of integer range
                    bookmarks: -1, // Negative bookmarks
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Out of range values",
                        }],
                    },
                },
            },
            businessLogic: {
                selfParentReference: {
                    id: commentDbIds.comment2,
                    score: 0,
                    bookmarks: 0,
                    parentId: commentDbIds.comment2, // Self-reference
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Self-referencing comment",
                        }],
                    },
                },
                circularParentReference: {
                    id: commentDbIds.comment3,
                    score: 0,
                    bookmarks: 0,
                    parentId: commentDbIds.comment4, // Circular reference setup
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Circular reference comment",
                        }],
                    },
                },
                noOwnershipOrParentObject: {
                    id: generatePK(),
                    score: 0,
                    bookmarks: 0,
                    // No ownedByUser, ownedByTeam, or parent object relations
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Comment with no ownership or parent",
                        }],
                    },
                },
            },
        };
    }

    /**
     * Generate fresh identifiers for comments
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            // Comments don't have publicId or handle
        };
    }

    /**
     * Add authentication context to a comment fixture
     */
    protected addAuthentication(data: Prisma.commentUncheckedCreateInput): Prisma.commentUncheckedCreateInput {
        return {
            ...data,
            ownedByUserId: commentDbIds.user1,
        };
    }

    /**
     * Add team context to a comment fixture
     */
    protected addTeamMemberships(data: Prisma.commentUncheckedCreateInput, teams: Array<{ teamId: string; role: string }>): Prisma.commentUncheckedCreateInput {
        // For comments, we set team ownership instead of membership
        if (teams.length > 0) {
            return {
                ...data,
                ownedByTeamId: BigInt(teams[0].teamId),
            };
        }
        return data;
    }

    /**
     * Add other relations to a comment fixture
     */
    protected addOtherRelations(data: Prisma.commentUncheckedCreateInput): Prisma.commentUncheckedCreateInput {
        return {
            ...data,
            issueId: commentDbIds.issue1,
            reactions: {
                create: [{
                    id: generatePK(),
                    emoji: "👍",
                    byId: commentDbIds.user1,
                }],
            },
        };
    }

    /**
     * Comment-specific validation
     */
    protected validateSpecific(data: Prisma.commentUncheckedCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Comment
        if (!data.translations) {
            errors.push("Comment translations are required");
        } else if (data.translations.create && Array.isArray(data.translations.create) && data.translations.create.length === 0) {
            errors.push("Comment must have at least one translation");
        }

        // Check business logic
        if (data.ownedByUserId && data.ownedByTeamId) {
            warnings.push("Comment should not be owned by both user and team");
        }

        if (!data.ownedByUserId && !data.ownedByTeamId && !data.parentId) {
            warnings.push("Comment should have ownership or be a reply to another comment");
        }

        // Check for parent object relationships
        const parentObjects = [data.issueId, data.resourceVersionId, data.pullRequestId].filter(Boolean);
        if (parentObjects.length === 0 && !data.parentId) {
            warnings.push("Comment should be associated with a parent object or be a reply");
        }

        if (parentObjects.length > 1) {
            warnings.push("Comment should typically be associated with only one parent object");
        }

        // Check score and bookmark values
        if (typeof data.score === "number" && data.score < -1000000) {
            warnings.push("Extremely negative score may indicate data issues");
        }

        if (typeof data.bookmarks === "number" && data.bookmarks < 0) {
            errors.push("Bookmark count cannot be negative");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.commentUncheckedCreateInput>): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        return factory.createMinimal(overrides);
    }

    static createWithTranslations(
        translations: Array<{ language: string; text: string }>,
        overrides?: Partial<Prisma.commentUncheckedCreateInput>,
    ): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        return factory.createMinimal({
            ...overrides,
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    text: t.text,
                })),
            },
        });
    }

    static createReply(
        parentId: string,
        overrides?: Partial<Prisma.commentUncheckedCreateInput>,
    ): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        return factory.createMinimal({
            ...overrides,
            parentId: BigInt(parentId),
        });
    }

    static createThreaded(
        createdById: string,
        parentId?: string,
        overrides?: Partial<Prisma.commentUncheckedCreateInput>,
    ): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        return factory.createMinimal({
            ...overrides,
            ownedByUserId: BigInt(createdById),
            ...(parentId && { parentId: BigInt(parentId) }),
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: overrides?.translations?.create?.[0]?.text || "Test comment",
                }],
            },
        });
    }

    static createComplete(overrides?: Partial<Prisma.commentUncheckedCreateInput>): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        return factory.createComplete(overrides);
    }

    static createWithOwner(
        ownerId: string,
        ownerType: "user" | "team" = "user",
        overrides?: Partial<Prisma.commentUncheckedCreateInput>,
    ): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        const ownerRelation = ownerType === "user" 
            ? { ownedByUserId: BigInt(ownerId) }
            : { ownedByTeamId: BigInt(ownerId) };
        
        return factory.createMinimal({
            ...overrides,
            ...ownerRelation,
        });
    }

    static createWithParentObject(
        objectId: string,
        objectType: "issue" | "resourceVersion" | "pullRequest",
        overrides?: Partial<Prisma.commentUncheckedCreateInput>,
    ): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        const parentRelation = {
            [`${objectType}Id`]: BigInt(objectId),
        };
        
        return factory.createMinimal({
            ...overrides,
            ...parentRelation,
        });
    }

    /**
     * Create comment for a specific object type using the commentFor pattern
     */
    static createForObject(
        objectType: CommentFor,
        objectId: string,
        overrides?: Partial<Prisma.commentUncheckedCreateInput>,
    ): Prisma.commentUncheckedCreateInput {
        const factory = new CommentDbFactory();
        const typeMapping = {
            [CommentFor.Issue]: "issueId",
            [CommentFor.PullRequest]: "pullRequestId",
            [CommentFor.ResourceVersion]: "resourceVersionId",
        };
        
        const relationField = typeMapping[objectType];
        if (!relationField) {
            throw new Error(`Invalid comment object type: ${objectType}`);
        }
        
        return factory.createMinimal({
            ...overrides,
            [relationField]: BigInt(objectId),
        });
    }

    /**
     * Create bulk comments for various object types
     */
    static createBulkForObjects(
        objects: Array<{ type: CommentFor; id: string; userId: string; text?: string }>,
    ): Prisma.commentUncheckedCreateInput[] {
        return objects.map((obj, index) => 
            this.createForObject(obj.type, obj.id, {
                ownedByUserId: BigInt(obj.userId),
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: obj.text || `Comment ${index + 1} on ${obj.type}`,
                    }],
                },
            }),
        );
    }
}

/**
 * Enhanced helper to seed a comment thread with comprehensive options
 */
export async function seedCommentThread(
    prisma: any,
    options: {
        createdById: string;
        objectId: string;
        objectType: "issue" | "resourceVersion" | "pullRequest";
        commentCount?: number;
        withReplies?: boolean;
        withReactions?: boolean;
        withBookmarks?: boolean;
    },
): Promise<BulkSeedResult<any>> {
    const factory = new CommentDbFactory();
    const comments = [];
    const count = options.commentCount || 3;
    let reactionsCount = 0;
    let bookmarksCount = 0;

    // Create root comments
    for (let i = 0; i < count; i++) {
        const commentData = CommentDbFactory.createThreaded(options.createdById, undefined, {
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: `Comment ${i + 1} - ${options.objectType}`,
                }],
            },
            // Connect to the object based on type
            [`${options.objectType}Id`]: BigInt(options.objectId),
            // Add reactions if requested
            ...(options.withReactions && {
                reactions: {
                    create: [{
                        id: generatePK(),
                        emoji: i % 2 === 0 ? "👍" : "❤️",
                        byId: BigInt(options.createdById),
                    }],
                },
            }),
            // Add bookmarks if requested
            ...(options.withBookmarks && {
                bookmarkedBy: {
                    create: [{
                        id: generatePK(),
                        userId: BigInt(options.createdById),
                    }],
                },
            }),
        });

        const comment = await prisma.comment.create({ data: commentData });
        comments.push(comment);

        if (options.withReactions) reactionsCount++;
        if (options.withBookmarks) bookmarksCount++;

        // Add replies if requested
        if (options.withReplies && i === 0) {
            for (let j = 0; j < 2; j++) {
                const replyData = CommentDbFactory.createThreaded(options.createdById, comment.id, {
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: `Reply ${j + 1} to Comment ${i + 1}`,
                        }],
                    },
                });
                
                const reply = await prisma.comment.create({ data: replyData });
                comments.push(reply);
            }
        }
    }

    return {
        records: comments,
        summary: {
            total: comments.length,
            withAuth: comments.length, // All comments have owners
            bots: 0, // Comments don't have bot concept
            teams: 0, // Not tracking team ownership in this helper
        },
    };
}

/**
 * Enhanced helper to seed multiple test comments with comprehensive options
 */
export async function seedTestComments(
    prisma: any,
    count = 5,
    options?: BulkSeedOptions & {
        parentObjectId?: string;
        parentObjectType?: "issue" | "resourceVersion" | "pullRequest";
        withReplies?: boolean;
        withReactions?: boolean;
        nestedLevel?: number;
    },
): Promise<BulkSeedResult<any>> {
    const factory = new CommentDbFactory();
    const comments = [];
    let totalWithAuth = 0;
    let totalTeams = 0;

    for (let i = 0; i < count; i++) {
        const overrides = options?.overrides?.[i] || {};
        let commentData: Prisma.commentUncheckedCreateInput;

        if (options?.withAuth) {
            commentData = factory.createWithRelationships({
                withAuth: true,
                overrides: {
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: `Test comment ${i + 1}`,
                        }],
                    },
                    ...overrides,
                },
            }).data;
            totalWithAuth++;
        } else {
            commentData = factory.createMinimal({
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: `Test comment ${i + 1}`,
                    }],
                },
                ...overrides,
            });
        }

        // Add parent object relationship if specified
        if (options?.parentObjectId && options?.parentObjectType) {
            commentData[`${options.parentObjectType}Id`] = BigInt(options.parentObjectId);
        }

        // Add team ownership if requested
        if (options?.teamId) {
            commentData.ownedByTeamId = BigInt(options.teamId);
            totalTeams++;
        }

        // Add reactions if requested
        if (options?.withReactions) {
            commentData.reactions = {
                create: [{
                    id: generatePK(),
                    emoji: ["👍", "❤️", "😄", "😮", "😢", "😡"][i % 6],
                    byId: commentDbIds.user1,
                }],
            };
        }

        const comment = await prisma.comment.create({ data: commentData });
        comments.push(comment);

        // Create nested replies if requested
        if (options?.withReplies && options?.nestedLevel && options.nestedLevel > 0) {
            let parentComment = comment;
            for (let level = 0; level < options.nestedLevel; level++) {
                const replyData = factory.createMinimal({
                    parentId: BigInt(parentComment.id),
                    ownedByUserId: commentDbIds.user1,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: `Nested reply level ${level + 1} to comment ${i + 1}`,
                        }],
                    },
                });
                
                const reply = await prisma.comment.create({ data: replyData });
                comments.push(reply);
                parentComment = reply;
            }
        }
    }

    return {
        records: comments,
        summary: {
            total: comments.length,
            withAuth: totalWithAuth,
            bots: 0, // Comments don't have bot concept
            teams: totalTeams,
        },
    };
}
