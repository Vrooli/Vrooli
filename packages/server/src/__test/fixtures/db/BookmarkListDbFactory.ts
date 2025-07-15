import { nanoid } from "@vrooli/shared";
import { type bookmark_list } from "@prisma/client";
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
    bookmark_list,
    Prisma.bookmark_listCreateInput,
    Prisma.bookmark_listInclude,
    Prisma.bookmark_listUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};

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
                id: this.generateId(),
                label: `Bookmarks_${nanoid()}`,
                user: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                label: "My Favorite Resources",
                user: { connect: { id: this.generateId() } },
                bookmarks: {
                    create: [
                        {
                            id: this.generateId(),
                            resourceId: this.generateId(),
                        },
                        {
                            id: this.generateId(),
                            resourceId: this.generateId(),
                        },
                        {
                            id: this.generateId(),
                            commentId: this.generateId(),
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
                    id: this.generateId(),
                    // Missing label
                    user: { connect: { id: this.generateId() } },
                },
                missingUserId: {
                    id: this.generateId(),
                    label: "Orphaned List",
                    // Missing userId
                },
                duplicateLabel: {
                    id: this.generateId(),
                    label: "existing_label", // Assumes this exists for the same user
                    user: { connect: { id: BigInt("1234567890") } }, // Fixed ID for testing duplicate detection
                },
                exceedsLabelLength: {
                    id: this.generateId(),
                    label: "a".repeat(129), // Exceeds 128 character limit
                    user: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                emptyList: {
                    id: this.generateId(),
                    label: "Empty List",
                    user: { connect: { id: this.generateId() } },
                    // No bookmarks
                },
                maxLengthLabel: {
                    id: this.generateId(),
                    label: "a".repeat(128), // Max length label
                    user: { connect: { id: this.generateId() } },
                },
                unicodeLabel: {
                    id: this.generateId(),
                    label: "📚 My Reading List 🌟",
                    user: { connect: { id: this.generateId() } },
                },
                specialCharactersLabel: {
                    id: this.generateId(),
                    label: "List with @#$% special chars!",
                    user: { connect: { id: this.generateId() } },
                },
                largeBookmarkCollection: {
                    id: this.generateId(),
                    label: "Large Collection",
                    user: { connect: { id: this.generateId() } },
                    bookmarks: {
                        create: Array.from({ length: 100 }, () => ({
                            id: this.generateId(),
                            resourceId: this.generateId(),
                        })),
                    },
                },
                mixedBookmarkTypes: {
                    id: this.generateId(),
                    label: "Mixed Types",
                    user: { connect: { id: this.generateId() } },
                    bookmarks: {
                        create: [
                            { id: this.generateId(), resourceId: this.generateId() },
                            { id: this.generateId(), commentId: this.generateId() },
                            { id: this.generateId(), issueId: this.generateId() },
                            { id: this.generateId(), tagId: this.generateId() },
                            { id: this.generateId(), teamId: this.generateId() },
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
                                id: this.generateId(),
                                resourceId: this.generateId(),
                            },
                        ],
                        deleteMany: {
                            listId: this.generateId(),
                        },
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        const uniqueLabel = `List_${nanoid()}`;
        
        return {
            id: this.generateId(),
            label: uniqueLabel,
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        const uniqueLabel = `Complete_List_${nanoid()}`;
        
        return {
            id: this.generateId(),
            label: uniqueLabel,
            user: { connect: { id: this.generateId() } },
            bookmarks: {
                create: [
                    {
                        id: this.generateId(),
                        resourceId: this.generateId(),
                    },
                    {
                        id: this.generateId(),
                        resourceId: this.generateId(),
                    },
                    {
                        id: this.generateId(),
                        commentId: this.generateId(),
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
    async createWithBookmarks(userId: string | bigint, label: string, bookmarkCount: number) {
        const bookmarks = Array.from({ length: bookmarkCount }, () => ({
            id: this.generateId(),
            resourceId: this.generateId(),
        }));

        return await this.createMinimal({
            user: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
            label,
            bookmarks: {
                create: bookmarks,
            },
        });
    }

    /**
     * Create multiple lists for a user
     */
    async createUserLists(userId: string | bigint, count = 3) {
        const lists = [];
        
        for (let i = 0; i < count; i++) {
            const list = await this.createMinimal({
                user: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
                label: `List ${i + 1}`,
            });
            lists.push(list);
        }
        
        return lists;
    }

    /**
     * Create predefined list templates
     */
    async createReadingList(userId: string | bigint) {
        return await this.createMinimal({
            user: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
            label: "📚 Reading List",
        });
    }

    async createFavoritesList(userId: string | bigint) {
        return await this.createMinimal({
            user: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
            label: "⭐ Favorites",
        });
    }

    async createWatchLaterList(userId: string | bigint) {
        return await this.createMinimal({
            user: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
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
            const userId = typeof config.withUser === "string" ? BigInt(config.withUser) : this.generateId();
            data.user = { connect: { id: userId } };
        }

        // Handle bookmarks
        if (config.withBookmarks) {
            const bookmarkCount = typeof config.withBookmarks === "number" ? config.withBookmarks : 3;
            data.bookmarks = {
                create: Array.from({ length: bookmarkCount }, () => ({
                    id: this.generateId(),
                    resourceId: this.generateId(),
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: any): Promise<string[]> {
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
        record: bookmark_list,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete bookmarks in this list
        if (shouldDelete("bookmarks")) {
            await tx.bookmark.deleteMany({
                where: { listId: record.id },
            });
        }
    }

    /**
     * Check if a user already has a list with a given label
     */
    async hasListWithLabel(userId: string | bigint, label: string): Promise<boolean> {
        const list = await this.prisma.bookmark_list.findFirst({
            where: {
                userId: typeof userId === "string" ? BigInt(userId) : userId,
                label,
            },
        });
        
        return list !== null;
    }

    /**
     * Get all lists for a user
     */
    async getUserLists(userId: string | bigint, includeBookmarks = false) {
        return await this.prisma.bookmark_list.findMany({
            where: { userId: typeof userId === "string" ? BigInt(userId) : userId },
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
    async addBookmarkToList(listId: string | bigint, bookmarkData: Partial<Prisma.bookmarkUncheckedCreateInput>) {
        return await this.prisma.bookmark.create({
            data: {
                id: this.generateId(),
                listId: typeof listId === "string" ? BigInt(listId) : listId,
                ...bookmarkData,
            },
        });
    }

    /**
     * Move bookmarks between lists
     */
    async moveBookmarks(fromListId: string | bigint, toListId: string | bigint, bookmarkIds: (string | bigint)[]): Promise<number> {
        const result = await this.prisma.bookmark.updateMany({
            where: {
                id: { in: bookmarkIds.map(id => typeof id === "string" ? BigInt(id) : id) },
                listId: typeof fromListId === "string" ? BigInt(fromListId) : fromListId,
            },
            data: {
                listId: typeof toListId === "string" ? BigInt(toListId) : toListId,
            },
        });
        
        return result.count;
    }
}

// Export factory creator function
export const createBookmarkListDbFactory = (prisma: PrismaClient) => 
    new BookmarkListDbFactory(prisma);

// Export the class for type usage
export { BookmarkListDbFactory as BookmarkListDbFactoryClass };
