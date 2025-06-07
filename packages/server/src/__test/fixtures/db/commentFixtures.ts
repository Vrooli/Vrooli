import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Comment model - used for seeding test data
 */

// Consistent IDs for testing
export const commentDbIds = {
    comment1: generatePK(),
    comment2: generatePK(),
    comment3: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
};

/**
 * Factory for creating comment database fixtures
 */
export class CommentDbFactory {
    static createMinimal(overrides?: Partial<Prisma.CommentCreateInput>): Prisma.CommentCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            threadId: generatePK(),
            ...overrides,
        };
    }

    static createWithTranslations(
        translations: Array<{ language: string; text: string }>,
        overrides?: Partial<Prisma.CommentCreateInput>
    ): Prisma.CommentCreateInput {
        return {
            ...this.createMinimal(overrides),
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    text: t.text,
                })),
            },
        };
    }

    static createReply(
        parentId: string,
        overrides?: Partial<Prisma.CommentCreateInput>
    ): Prisma.CommentCreateInput {
        return {
            ...this.createMinimal(overrides),
            parent: { connect: { id: parentId } },
        };
    }

    static createThreaded(
        createdById: string,
        parentId?: string,
        overrides?: Partial<Prisma.CommentCreateInput>
    ): Prisma.CommentCreateInput {
        return {
            ...this.createMinimal(overrides),
            createdBy: { connect: { id: createdById } },
            ...(parentId && { parent: { connect: { id: parentId } } }),
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: overrides?.translations?.create?.[0]?.text || "Test comment",
                }],
            },
        };
    }
}

/**
 * Helper to seed a comment thread
 */
export async function seedCommentThread(
    prisma: any,
    options: {
        createdById: string;
        objectId: string;
        objectType: string;
        commentCount?: number;
        withReplies?: boolean;
    }
) {
    const comments = [];
    const count = options.commentCount || 3;

    // Create root comments
    for (let i = 0; i < count; i++) {
        const comment = await prisma.comment.create({
            data: CommentDbFactory.createThreaded(options.createdById, undefined, {
                threadId: options.objectId,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: `Comment ${i + 1}`,
                    }],
                },
                // Connect to the object based on type
                ...(options.objectType === "Api" && { api: { connect: { id: options.objectId } } }),
                ...(options.objectType === "Code" && { code: { connect: { id: options.objectId } } }),
                ...(options.objectType === "Project" && { project: { connect: { id: options.objectId } } }),
                ...(options.objectType === "Routine" && { routine: { connect: { id: options.objectId } } }),
                ...(options.objectType === "Standard" && { standard: { connect: { id: options.objectId } } }),
                ...(options.objectType === "Team" && { team: { connect: { id: options.objectId } } }),
            }),
        });
        comments.push(comment);

        // Add replies if requested
        if (options.withReplies && i === 0) {
            for (let j = 0; j < 2; j++) {
                const reply = await prisma.comment.create({
                    data: CommentDbFactory.createThreaded(options.createdById, comment.id, {
                        threadId: options.objectId,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                text: `Reply ${j + 1} to Comment ${i + 1}`,
                            }],
                        },
                    }),
                });
                comments.push(reply);
            }
        }
    }

    return comments;
}