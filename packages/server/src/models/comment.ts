import { CODE, commentCreate, CommentFor, commentUpdate } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentCreateInput, CommentUpdateInput, DeleteOneInput, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountQueries, addCreatorField, addJoinTables, FormatConverter, MODEL_TYPES, removeCountQueries, removeCreatorField, removeJoinTables, selectHelper } from "./base";
import { hasProfanity } from "../utils/censor";
import { GraphQLResolveInfo } from "graphql";
import { OrganizationModel } from "./organization";
import { comment } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Comment, comment> => {
    const joinMapper = {
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Comment>): RecursivePartial<comment> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            // Remove isUpvoted and isStarred, as they are calculated in their own queries
            if (modified.isUpvoted) delete modified.isUpvoted;
            if (modified.isStarred) delete modified.isStarred;
            console.log('comment toDB', modified);
            // if (modified.commentedOn) {
            //     if (modified.creator.hasOwnProperty('username')) {
            //         modified.createdByUser = modified.creator;
            //     } else {
            //         modified.createdByOrganization = modified.creator;
            //     }
            //     delete modified.creator;
            // }
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<comment>): RecursivePartial<Comment> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            modified = addCreatorField(modified);
            if (modified.project) {
                modified.commentedOn = modified.project;
                delete modified.project;
            }
            else if (modified.routine) {
                modified.commentedOn = modified.routine;
                delete modified.routine;
            }
            else if (modified.standard) {
                modified.commentedOn = modified.standard;
                delete modified.standard;
            }
            return modified;
        },
    }
}

const forMapper = {
    [CommentFor.Project]: 'projectId',
    [CommentFor.Routine]: 'routineId',
    [CommentFor.Standard]: 'standardId',
}

/**
 * Handles the authorized adding, updating, and deleting of comments.
 * Only users can add comments, and they can do so multiple times on 
 * the same object.
 */
const commenter = (format: FormatConverter<Comment, comment>, prisma: PrismaType) => ({
    async create(
        userId: string,
        input: CommentCreateInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<Comment>> {
        // Check for valid arguments
        commentCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.text)) throw new CustomError(CODE.BannedWord);
        // Add comment
        let comment = await prisma.comment.create({
            data: {
                text: input.text,
                userId,
                [forMapper[input.createdFor]]: input.forId,
            },
            ...selectHelper<CommentCreateInput | CommentUpdateInput, comment>(info, formatter().toDB)
        });
        // Return comment with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.toGraphQL(comment), isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: CommentUpdateInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<Comment>> {
        // Check for valid arguments
        commentUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.text)) throw new CustomError(CODE.BannedWord);
        // Find comment
        let comment = await prisma.comment.findUnique({ where: { id: input.id } });
        if (!comment) throw new CustomError(CODE.NotFound, "Comment not found");
        // Update comment
        comment = await prisma.comment.update({
            where: { id: comment.id },
            data: {
                text: input.text,
            },
            ...selectHelper<CommentCreateInput | CommentUpdateInput, comment>(info, formatter().toDB)
        });
        // Return comment with "isUpvoted" and "isStarred" fields. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, commentId: comment.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, commentId: comment.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(comment), isUpvoted, isStarred };
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const comment = await prisma.comment.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                userId: true,
                organizationId: true,
            }
        })
        if (!comment) throw new CustomError(CODE.NotFound, "Comment not found");
        // Check if user is authorized
        let authorized = userId === comment.userId;
        if (!authorized && comment.organizationId) {
            authorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, comment.organizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.comment.delete({
            where: { id: comment.id },
        });
        return { success: true };
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Comment;
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...commenter(format, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================