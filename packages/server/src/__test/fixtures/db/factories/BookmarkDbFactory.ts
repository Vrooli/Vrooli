import { generatePK, generatePublicId, BookmarkFor } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface BookmarkRelationConfig extends RelationConfig {
    listId?: string;
    objectType?: string;
    objectId?: string;
}

/**
 * Database fixture factory for Bookmark model
 * Handles polymorphic bookmarking of any object with optional list organization
 */
export class BookmarkDbFactory extends DatabaseFixtureFactory<
    Prisma.Bookmark,
    Prisma.BookmarkCreateInput,
    Prisma.BookmarkInclude,
    Prisma.BookmarkUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Bookmark', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.bookmark;
    }

    protected getMinimalData(overrides?: Partial<Prisma.BookmarkCreateInput>): Prisma.BookmarkCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.BookmarkCreateInput>): Prisma.BookmarkCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            by: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            list: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    /**
     * Create a bookmark for a specific object type
     */
    async createForObject(
        byId: string,
        objectType: BookmarkFor | string,
        objectId: string,
        overrides?: Partial<Prisma.BookmarkCreateInput>
    ): Promise<Prisma.Bookmark> {
        const typeMapping: Record<string, string> = {
            // BookmarkFor enum values
            [BookmarkFor.Comment]: 'comment',
            [BookmarkFor.Issue]: 'issue',
            [BookmarkFor.Resource]: 'resource',
            [BookmarkFor.Tag]: 'tag',
            [BookmarkFor.Team]: 'team',
            [BookmarkFor.User]: 'user',
            // Additional supported types
            'Api': 'api',
            'Code': 'code',
            'Note': 'note',
            'Post': 'post',
            'Project': 'project',
            'Prompt': 'prompt',
            'Question': 'question',
            'Quiz': 'quiz',
            'Routine': 'routine',
            'RunProject': 'runProject',
            'RunRoutine': 'runRoutine',
            'SmartContract': 'smartContract',
            'Standard': 'standard',
        };

        const fieldName = typeMapping[objectType];
        if (!fieldName) {
            throw new Error(`Invalid bookmark object type: ${objectType}`);
        }

        const data: Prisma.BookmarkCreateInput = {
            ...this.getMinimalData(),
            by: { connect: { id: byId } },
            [fieldName]: { connect: { id: objectId } },
            ...overrides,
        };

        const result = await this.prisma.bookmark.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create a bookmark in a specific list
     */
    async createInList(
        byId: string,
        listId: string,
        objectType: BookmarkFor | string,
        objectId: string,
        overrides?: Partial<Prisma.BookmarkCreateInput>
    ): Promise<Prisma.Bookmark> {
        const bookmark = await this.createForObject(byId, objectType, objectId, {
            list: { connect: { id: listId } },
            ...overrides,
        });
        return bookmark;
    }

    /**
     * Create multiple bookmarks for a user
     */
    async createBulkForUser(
        byId: string,
        objects: Array<{ type: BookmarkFor | string; id: string }>,
        listId?: string
    ): Promise<Prisma.Bookmark[]> {
        const bookmarks: Prisma.Bookmark[] = [];
        
        for (const obj of objects) {
            const bookmark = listId
                ? await this.createInList(byId, listId, obj.type, obj.id)
                : await this.createForObject(byId, obj.type, obj.id);
            bookmarks.push(bookmark);
        }
        
        return bookmarks;
    }

    protected getDefaultInclude(): Prisma.BookmarkInclude {
        return {
            by: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            list: {
                select: {
                    id: true,
                    publicId: true,
                    translations: {
                        where: { language: 'en' },
                        take: 1,
                    },
                },
            },
            // Include polymorphic relationships
            api: { select: { id: true, publicId: true } },
            code: { select: { id: true, publicId: true } },
            comment: { select: { id: true } },
            issue: { select: { id: true, publicId: true } },
            note: { select: { id: true, publicId: true } },
            post: { select: { id: true, publicId: true } },
            project: { select: { id: true, publicId: true } },
            prompt: { select: { id: true, publicId: true } },
            question: { select: { id: true, publicId: true } },
            quiz: { select: { id: true, publicId: true } },
            resource: { select: { id: true, publicId: true } },
            routine: { select: { id: true, publicId: true } },
            runProject: { select: { id: true, publicId: true } },
            runRoutine: { select: { id: true, publicId: true } },
            smartContract: { select: { id: true, publicId: true } },
            standard: { select: { id: true, publicId: true } },
            tag: { select: { id: true, name: true } },
            team: { select: { id: true, publicId: true, handle: true } },
            user: { select: { id: true, publicId: true, name: true, handle: true } },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.BookmarkCreateInput,
        config: BookmarkRelationConfig,
        tx: any
    ): Promise<Prisma.BookmarkCreateInput> {
        let data = { ...baseData };

        // Ensure user is set
        if (!data.by && config.byId) {
            data.by = { connect: { id: config.byId } };
        }

        // Handle list connection
        if (config.listId) {
            data.list = { connect: { id: config.listId } };
        }

        // Handle object connection
        if (config.objectType && config.objectId) {
            const typeMapping: Record<string, string> = {
                [BookmarkFor.Comment]: 'comment',
                [BookmarkFor.Issue]: 'issue',
                [BookmarkFor.Resource]: 'resource',
                [BookmarkFor.Tag]: 'tag',
                [BookmarkFor.Team]: 'team',
                [BookmarkFor.User]: 'user',
                'Api': 'api',
                'Code': 'code',
                'Note': 'note',
                'Post': 'post',
                'Project': 'project',
                'Prompt': 'prompt',
                'Question': 'question',
                'Quiz': 'quiz',
                'Routine': 'routine',
                'RunProject': 'runProject',
                'RunRoutine': 'runRoutine',
                'SmartContract': 'smartContract',
                'Standard': 'standard',
            };

            const fieldName = typeMapping[config.objectType];
            if (fieldName) {
                data[fieldName] = { connect: { id: config.objectId } };
            }
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createUserBookmarksOwnProfile(userId: string): Promise<Prisma.Bookmark> {
        return this.createForObject(userId, BookmarkFor.User, userId, {
            // User bookmarking themselves (edge case)
        });
    }

    async createOrganizedCollection(
        userId: string,
        listId: string,
        theme: string
    ): Promise<Prisma.Bookmark[]> {
        // Create themed bookmarks
        const themedObjects = [
            { type: 'Project', id: generatePK() },
            { type: 'Routine', id: generatePK() },
            { type: 'Standard', id: generatePK() },
            { type: BookmarkFor.Resource, id: generatePK() },
        ];

        return this.createBulkForUser(userId, themedObjects, listId);
    }

    async createCrossTypeBookmarks(userId: string): Promise<Prisma.Bookmark[]> {
        // Create bookmarks across different object types
        const diverseObjects = [
            { type: BookmarkFor.User, id: generatePK() },
            { type: BookmarkFor.Team, id: generatePK() },
            { type: BookmarkFor.Comment, id: generatePK() },
            { type: BookmarkFor.Issue, id: generatePK() },
            { type: BookmarkFor.Resource, id: generatePK() },
            { type: BookmarkFor.Tag, id: generatePK() },
            { type: 'Project', id: generatePK() },
            { type: 'Routine', id: generatePK() },
        ];

        return this.createBulkForUser(userId, diverseObjects);
    }

    protected async checkModelConstraints(record: Prisma.Bookmark): Promise<string[]> {
        const violations: string[] = [];
        
        // Check publicId uniqueness
        if (record.publicId) {
            const duplicate = await this.prisma.bookmark.findFirst({
                where: { 
                    publicId: record.publicId,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push('PublicId must be unique');
            }
        }

        // Check user exists
        if (record.byId) {
            const user = await this.prisma.user.findUnique({
                where: { id: record.byId },
            });
            if (!user) {
                violations.push('Bookmark user must exist');
            }
        }

        // Check exactly one bookmarked object
        const bookmarkableFields = [
            'apiId', 'codeId', 'commentId', 'issueId', 'noteId', 'postId',
            'projectId', 'promptId', 'questionId', 'quizId', 'resourceId',
            'routineId', 'runProjectId', 'runRoutineId', 'smartContractId',
            'standardId', 'tagId', 'teamId', 'userId'
        ];
        
        const connectedObjects = bookmarkableFields.filter(field => 
            record[field as keyof Prisma.Bookmark]
        );
        
        if (connectedObjects.length === 0) {
            violations.push('Bookmark must reference exactly one object');
        } else if (connectedObjects.length > 1) {
            violations.push('Bookmark cannot reference multiple objects');
        }

        // Check list exists if specified
        if (record.listId) {
            const list = await this.prisma.bookmarkList.findUnique({
                where: { id: record.listId },
            });
            if (!list) {
                violations.push('Bookmark list must exist');
            } else if (list.createdById !== record.byId) {
                violations.push('Bookmark list must belong to the same user');
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
                // Missing by (user)
                publicId: generatePublicId(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                by: "invalid-user-reference", // Should be connect object
            },
            noBookmarkableObject: {
                id: generatePK(),
                publicId: generatePublicId(),
                by: { connect: { id: generatePK() } },
                // No object connected
            },
            multipleObjects: {
                id: generatePK(),
                publicId: generatePublicId(),
                by: { connect: { id: generatePK() } },
                project: { connect: { id: generatePK() } },
                routine: { connect: { id: generatePK() } },
                user: { connect: { id: generatePK() } },
                // Multiple objects (invalid)
            },
            invalidList: {
                id: generatePK(),
                publicId: generatePublicId(),
                by: { connect: { id: generatePK() } },
                list: { connect: { id: "999999999999999" } }, // Non-existent list
                project: { connect: { id: generatePK() } },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.BookmarkCreateInput> {
        const userId = generatePK();
        const projectId = generatePK();
        
        return {
            selfBookmark: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                user: { connect: { id: userId } }, // User bookmarking themselves
            },
            orphanedBookmark: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                project: { connect: { id: projectId } },
                // No list - floating bookmark
            },
            tagBookmark: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                tag: { connect: { id: generatePK() } }, // Less common bookmark type
            },
            commentBookmark: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                comment: { connect: { id: generatePK() } }, // Bookmarking a comment
            },
            runBookmark: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                runRoutine: { connect: { id: generatePK() } }, // Bookmarking a run
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            // No direct children to cascade
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Bookmark,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Bookmarks don't have child records to delete
    }
}

// Export factory creator function
export const createBookmarkDbFactory = (prisma: PrismaClient) => new BookmarkDbFactory(prisma);