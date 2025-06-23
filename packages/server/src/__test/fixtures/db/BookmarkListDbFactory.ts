import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface BookmarkListRelationConfig extends RelationConfig {
    withUser?: boolean | string;
    withBookmarks?: boolean | number;
}

/**
 * Enhanced database fixture factory for BookmarkList model
 * Provides comprehensive testing capabilities for bookmark list management
 * 
 * Features:
 * - Type-safe Prisma integration
 * - User-specific list management
 * - Bookmark collection handling
 * - Label uniqueness per user
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class BookmarkListDbFactory extends EnhancedDatabaseFactory<
    Prisma.bookmark_listCreateInput,
    Prisma.bookmark_listCreateInput,
    Prisma.bookmark_listInclude,
    Prisma.bookmark_listUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("bookmark_list", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.bookmark_list;
    }

    /**
     * Get complete test fixtures for BookmarkList model
     */
    protected getFixtures(): DbTestFixtures<Prisma.bookmark_listCreateInput, Prisma.bookmark_listUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                label: `Bookmarks_${nanoid(8)}`,
                userId: generatePK().toString(),
            },
            complete: {
                id: generatePK().toString(),
                label: "My Favorite Resources",
                userId: generatePK().toString(),
                bookmarks: {
                    create: [
                        {
                            id: generatePK().toString(),
                            resourceId: generatePK().toString(),
                        },
                        {
                            id: generatePK().toString(),
                            resourceId: generatePK().toString(),
                        },
                        {
                            id: generatePK().toString(),
                            commentId: generatePK().toString(),
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, label, and userId
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    label: 123, // Should be string
                    userId: true, // Should be string
                },
                missingLabel: {
                    id: generatePK().toString(),
                    // Missing label
                    userId: generatePK().toString(),
                },
                missingUserId: {
                    id: generatePK().toString(),
                    label: "Orphaned List",
                    // Missing userId
                },
                duplicateLabel: {
                    id: generatePK().toString(),
                    label: "existing_label", // Assumes this exists for the same user
                    userId: "existing_user_id",
                },
                exceedsLabelLength: {
                    id: generatePK().toString(),
                    label: "a".repeat(129), // Exceeds 128 character limit
                    userId: generatePK().toString(),
                },
            },
            edgeCases: {
                emptyList: {
                    id: generatePK().toString(),
                    label: "Empty List",
                    userId: generatePK().toString(),
                    // No bookmarks
                },
                maxLengthLabel: {
                    id: generatePK().toString(),
                    label: "a".repeat(128), // Max length label
                    userId: generatePK().toString(),
                },
                unicodeLabel: {
                    id: generatePK().toString(),
                    label: "📚 My Reading List 🌟",
                    userId: generatePK().toString(),
                },
                specialCharactersLabel: {
                    id: generatePK().toString(),
                    label: "List with @#$% special chars!",
                    userId: generatePK().toString(),
                },
                largeBookmarkCollection: {
                    id: generatePK().toString(),
                    label: "Large Collection",
                    userId: generatePK().toString(),
                    bookmarks: {
                        create: Array.from({ length: 100 }, () => ({
                            id: generatePK().toString(),
                            resourceId: generatePK().toString(),
                        })),
                    },
                },
                mixedBookmarkTypes: {
                    id: generatePK().toString(),
                    label: "Mixed Types",
                    userId: generatePK().toString(),
                    bookmarks: {
                        create: [
                            { id: generatePK().toString(), resourceId: generatePK().toString() },
                            { id: generatePK().toString(), commentId: generatePK().toString() },
                            { id: generatePK().toString(), issueId: generatePK().toString() },
                            { id: generatePK().toString(), tagId: generatePK().toString() },
                            { id: generatePK().toString(), teamId: generatePK().toString() },
                        ],
                    },
                },
            },
            updates: {
                minimal: {
                    label: "Updated Label",
                },
                complete: {
                    label: "Completely Updated List",
                    bookmarks: {
                        create: [
                            {
                                id: generatePK().toString(),
                                resourceId: generatePK().toString(),
                            },
                        ],
                        deleteMany: {
                            listId: generatePK().toString(),
                        },
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        const uniqueLabel = `List_${nanoid(8)}`;
        
        return {
            id: generatePK().toString(),
            label: uniqueLabel,
            userId: generatePK().toString(),
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        const uniqueLabel = `Complete_List_${nanoid(8)}`;
        
        return {
            id: generatePK().toString(),
            label: uniqueLabel,
            userId: generatePK().toString(),
            bookmarks: {
                create: [
                    {
                        id: generatePK().toString(),
                        resourceId: generatePK().toString(),
                    },
                    {
                        id: generatePK().toString(),
                        resourceId: generatePK().toString(),
                    },
                    {
                        id: generatePK().toString(),
                        commentId: generatePK().toString(),
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            emptyUserList: {
                name: "emptyUserList",
                description: "Empty bookmark list for a user",
                config: {
                    overrides: {
                        label: "My Reading List",
                    },
                },
            },
            resourceCollection: {
                name: "resourceCollection",
                description: "List with multiple resource bookmarks",
                config: {
                    overrides: {
                        label: "Useful Resources",
                    },
                    withBookmarks: 5,
                },
            },
            curatedList: {
                name: "curatedList",
                description: "Curated list with mixed bookmark types",
                config: {
                    overrides: {
                        label: "Curated Content",
                    },
                    withBookmarks: true,
                },
            },
            sharedTeamList: {
                name: "sharedTeamList",
                description: "List that could be shared with team members",
                config: {
                    overrides: {
                        label: "Team Resources",
                    },
                    withBookmarks: 10,
                },
            },
            temporaryList: {
                name: "temporaryList",
                description: "Temporary list for quick saves",
                config: {
                    overrides: {
                        label: "Quick Saves",
                    },
                    withBookmarks: 3,
                },
            },
        };
    }

    /**
     * Create a list with a specific number of bookmarks
     */
    async createWithBookmarks(userId: string, label: string, bookmarkCount: number): Promise<Prisma.bookmark_list> {
        const bookmarks = Array.from({ length: bookmarkCount }, () => ({
            id: generatePK().toString(),
            resourceId: generatePK().toString(),
        }));

        return await this.createMinimal({
            userId,
            label,
            bookmarks: {
                create: bookmarks,
            },
        });
    }

    /**
     * Create multiple lists for a user
     */
    async createUserLists(userId: string, count = 3): Promise<Prisma.bookmark_list[]> {
        const lists: Prisma.bookmark_list[] = [];
        
        for (let i = 0; i < count; i++) {
            const list = await this.createMinimal({
                userId,
                label: `List ${i + 1}`,
            });
            lists.push(list);
        }
        
        return lists;
    }

    /**
     * Create predefined list templates
     */
    async createReadingList(userId: string): Promise<Prisma.bookmark_list> {
        return await this.createMinimal({
            userId,
            label: "📚 Reading List",
        });
    }

    async createFavoritesList(userId: string): Promise<Prisma.bookmark_list> {
        return await this.createMinimal({
            userId,
            label: "⭐ Favorites",
        });
    }

    async createWatchLaterList(userId: string): Promise<Prisma.bookmark_list> {
        return await this.createMinimal({
            userId,
            label: "⏰ Watch Later",
        });
    }

    protected getDefaultInclude(): Prisma.bookmark_listInclude {
        return {
            bookmarks: {
                include: {
                    resource: true,
                    comment: true,
                    issue: true,
                    tag: true,
                    team: true,
                    user: true,
                },
            },
            user: true,
            _count: {
                select: {
                    bookmarks: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.bookmark_listCreateInput,
        config: BookmarkListRelationConfig,
        tx: any,
    ): Promise<Prisma.bookmark_listCreateInput> {
        const data = { ...baseData };

        // Handle user relationship
        if (config.withUser) {
            const userId = typeof config.withUser === "string" ? config.withUser : generatePK().toString();
            data.userId = userId;
        }

        // Handle bookmarks
        if (config.withBookmarks) {
            const bookmarkCount = typeof config.withBookmarks === "number" ? config.withBookmarks : 3;
            data.bookmarks = {
                create: Array.from({ length: bookmarkCount }, () => ({
                    id: generatePK().toString(),
                    resourceId: generatePK().toString(),
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.bookmark_list): Promise<string[]> {
        const violations: string[] = [];
        
        // Check label length
        if (record.label && record.label.length > 128) {
            violations.push("Label exceeds maximum length of 128 characters");
        }

        // Check label uniqueness per user
        if (record.label && record.userId) {
            const duplicate = await this.prisma.bookmark_list.findFirst({
                where: {
                    label: record.label,
                    userId: record.userId,
                    id: { not: record.id },
                },
            });
            
            if (duplicate) {
                violations.push("User already has a list with this label");
            }
        }

        // Check that list belongs to a user
        if (!record.userId) {
            violations.push("Bookmark list must belong to a user");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            bookmarks: true,
            user: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.bookmark_list,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete bookmarks in this list
        if (shouldDelete("bookmarks") && record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { listId: record.id },
            });
        }
    }

    /**
     * Check if a user already has a list with a given label
     */
    async hasListWithLabel(userId: string, label: string): Promise<boolean> {
        const list = await this.prisma.bookmark_list.findFirst({
            where: {
                userId,
                label,
            },
        });
        
        return list !== null;
    }

    /**
     * Get all lists for a user
     */
    async getUserLists(userId: string, includeBookmarks = false): Promise<Prisma.bookmark_list[]> {
        return await this.prisma.bookmark_list.findMany({
            where: { userId },
            include: {
                bookmarks: includeBookmarks,
                _count: {
                    select: {
                        bookmarks: true,
                    },
                },
            },
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Add a bookmark to a list
     */
    async addBookmarkToList(listId: string, bookmarkData: Partial<Prisma.bookmarkCreateInput>): Promise<Prisma.bookmark> {
        return await this.prisma.bookmark.create({
            data: {
                id: generatePK().toString(),
                listId,
                ...bookmarkData,
            },
        });
    }

    /**
     * Move bookmarks between lists
     */
    async moveBookmarks(fromListId: string, toListId: string, bookmarkIds: string[]): Promise<number> {
        const result = await this.prisma.bookmark.updateMany({
            where: {
                id: { in: bookmarkIds },
                listId: fromListId,
            },
            data: {
                listId: toListId,
            },
        });
        
        return result.count;
    }
}

// Export factory creator function
export const createBookmarkListDbFactory = (prisma: PrismaClient) => 
    BookmarkListDbFactory.getInstance("bookmark_list", prisma);

// Export the class for type usage
export { BookmarkListDbFactory as BookmarkListDbFactoryClass };
