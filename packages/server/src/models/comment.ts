import { CODE, CommentFor } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentInput, DeleteOneInput, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountQueries, addCreatorField, addJoinTables, FormatConverter, MODEL_TYPES, removeCountQueries, removeCreatorField, removeJoinTables, selectHelper } from "./base";
import { hasProfanity } from "../utils/censor";
import { GraphQLResolveInfo } from "graphql";
import { UserDB } from "./user";
import { OrganizationDB, OrganizationModel } from "./organization";
import { ProjectDB } from "./project";
import { RoutineDB } from "./routine";
import { StandardDB } from "./standard";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type CommentRelationshipList = 'user' | 'organization' | 'project' |
    'routine' | 'standard' | 'reports' | 'stars';
// Type 2. QueryablePrimitives
export type CommentQueryablePrimitives = Omit<Comment, CommentRelationshipList>;
// Type 3. AllPrimitives
export type CommentAllPrimitives = CommentQueryablePrimitives;
// type 4. Database shape
export type CommentDB = CommentAllPrimitives &
{
    user: UserDB,
    organization: OrganizationDB,
    project: ProjectDB,
    routine: RoutineDB,
    standard: StandardDB,
    stars: number,
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Comment, CommentDB> => {
    const joinMapper = {
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Comment>): RecursivePartial<CommentDB> => {
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
        toGraphQL: (obj: RecursivePartial<CommentDB>): RecursivePartial<Comment> => {
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
    [CommentFor.Project]: 'project',
    [CommentFor.Routine]: 'routine',
    [CommentFor.Standard]: 'standard',
}

/**
 * Handles the authorized adding, updating, and deleting of comments.
 * Only users can add comments, and they can do so multiple times on 
 * the same object.
 */
const commenter = (prisma: PrismaType) => ({
    async addComment(
        userId: string,
        input: CommentInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.text || input.text.length < 1) throw new CustomError(CODE.InternalError, 'Text is too short');
        // Check for censored words
        if (hasProfanity(input.text)) throw new CustomError(CODE.BannedWord);
        // Add comment
        let comment = await prisma.comment.create({
            data: {
                text: input.text,
                userId,
                [forMapper[input.createdFor]]: input.forId,
            },
            ...selectHelper<Comment, CommentDB>(info, formatter().toDB)
        });
        // Return comment with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...comment, isUpvoted: null, isStarred: false };
    },
    async updateComment(
        userId: string,
        input: CommentInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.text || input.text.length < 1) throw new CustomError(CODE.InternalError, 'Text too short');
        // Check for censored words
        if (hasProfanity(input.text)) throw new CustomError(CODE.BannedWord);
        // Find comment
        let comment = await prisma.comment.findFirst({
            where: {
                userId,
                [forMapper[input.createdFor]]: input.forId,
            }
        })
        if (!comment) throw new CustomError(CODE.NotFound, "Comment not found");
        // Update comment
        comment = await prisma.comment.update({
            where: { id: comment.id },
            data: {
                text: input.text,
            },
            ...selectHelper<Comment, CommentDB>(info, formatter().toDB)
        });
        // Return comment with "isUpvoted" and "isStarred" fields. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, commentId: comment.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, commentId: comment.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...comment, isUpvoted, isStarred };
    },
    async deleteComment(userId: string, input: DeleteOneInput): Promise<Success> {
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
        ...commenter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================