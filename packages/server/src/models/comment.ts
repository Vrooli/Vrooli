import { CODE, CommentFor } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentInput } from "schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountQueries, addCreatorField, addJoinTables, findByIder, FormatConverter, MODEL_TYPES, removeCountQueries, removeCreatorField, removeJoinTables, selectHelper } from "./base";
import { hasProfanity } from "../utils/censor";
import { GraphQLResolveInfo } from "graphql";

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
    Pick<Comment, 'user' | 'organization' | 'project' | 'routine' | 'standard' | 'reports'> &
{
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
const commenter = (prisma?: PrismaType) => ({
    async addComment(
        userId: string, 
        input: CommentInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!prisma || !input.text || input.text.length < 1) throw new CustomError(CODE.InvalidArgs);
        // Check for censored words
        if (hasProfanity(input.text)) throw new CustomError(CODE.BannedWord);
        // Add comment
        return await prisma.comment.create({
            data: {
                text: input.text,
                userId,
                [forMapper[input.createdFor]]: input.forId,
            },
            ...selectHelper<Comment, CommentDB>(info, formatter().toDB)
        });
    },
    async updateComment(
        userId: string, 
        input: CommentInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!prisma || !input.text || input.text.length < 1) throw new CustomError(CODE.InvalidArgs);
        // Check for censored words
        if (hasProfanity(input.text)) throw new CustomError(CODE.BannedWord);
        // Find comment
        const comment = await prisma.comment.findFirst({
            where: {
                userId,
                [forMapper[input.createdFor]]: input.forId,
            }
        })
        if (!comment) throw new CustomError(CODE.ErrorUnknown);
        // Update comment
        return await prisma.comment.update({
            where: { id: comment.id },
            data: {
                text: input.text,
            },
            ...selectHelper<Comment, CommentDB>(info, formatter().toDB)
        });
    },
    async deleteComment(userId: string, input: any): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Find comment
        const comment = await prisma.comment.findFirst({
            where: {
                userId,
                [forMapper[input.createdFor]]: input.forId,
            }
        })
        if (!comment) throw new CustomError(CODE.ErrorUnknown);
        // Delete comment
        await prisma.comment.delete({
            where: { id: comment.id },
        });
        return true;
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Comment;
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...commenter(prisma),
        ...findByIder<Comment, CommentDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================