import { CODE, commentCreate, CommentFor, commentUpdate } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentCreateInput, CommentUpdateInput, DeleteOneInput, Success } from "../schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { addCreatorField, addJoinTables, deconstructUnion, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, relationshipFormatter, removeCreatorField, removeJoinTables, selectHelper } from "./base";
import { hasProfanity } from "../utils/censor";
import { GraphQLResolveInfo } from "graphql";
import { OrganizationModel } from "./organization";
import { comment } from "@prisma/client";
import { routineDBFields } from "./routine";
import _ from "lodash";
import { projectDBFields } from "./project";
import { standardDBFields } from "./standard";

export const commentDBFields = ['id', 'text', 'created_at', 'updated_at', 'userId', 'organizationId', 'projectId', 'routineId', 'standardId', 'score', 'stars']

//==============================================================
/* #region Custom Components */
//==============================================================

type CommentFormatterType = FormatConverter<Comment, comment>;
/**
 * Component for formatting between graphql and prisma types
 */
export const commentFormatter = (): CommentFormatterType => ({
    joinMapper: ({
        starredBy: 'user',
    }),
    dbShape: (partial: PartialSelectConvert<Comment>): PartialSelectConvert<comment> => {
        let modified = partial;
        // Deconstruct GraphQL unions
        modified = removeCreatorField(modified);
        modified = deconstructUnion(modified, 'commentedOn', [
            ['project', {
                ...Object.fromEntries(projectDBFields.map(f => [f, true]))
            }],
            ['routine', {
                ...Object.fromEntries(routineDBFields.map(f => [f, true]))
            }],
            ['standard', {
                ...Object.fromEntries(standardDBFields.map(f => [f, true]))
            }],
        ]);
        // Convert relationships
        modified = relationshipFormatter(modified, [
            ['reports', FormatterMap.Report.dbShape],
            ['starredBy', FormatterMap.User.dbShapeUser],
        ]);
        // Add join tables not present in GraphQL type, but present in Prisma
        modified = addJoinTables(modified, commentFormatter().joinMapper);
        return modified;
    },
    dbPrune: (info: InfoType): PartialSelectConvert<comment> => {
        // Convert GraphQL info object to a partial select object
        let modified = infoToPartialSelect(info);
        // Remove calculated fields
        let { isUpvoted, isStarred, ...rest } = modified;
        modified = rest;
        // Convert relationships
        modified = relationshipFormatter(modified, [
            ['reports', FormatterMap.Report.dbPrune],
            ['starredBy', FormatterMap.User.dbPruneUser],
        ]);
        return modified;
    },
    selectToDB: (info: InfoType): PartialSelectConvert<comment> => {
        return commentFormatter().dbShape(commentFormatter().dbPrune(info));
    },
    selectToGraphQL: (obj: RecursivePartial<comment>): RecursivePartial<Comment> => {
        if (!_.isObject(obj)) return obj;
        // Create unions
        let { project, routine, standard, ...modified } = addCreatorField(obj);
        if (project) modified.commentedOn = FormatterMap.Project.selectToGraphQL(modified.project);
        else if (routine) modified.commentedOn = FormatterMap.Routine.selectToGraphQL(modified.routine);
        else if (standard) modified.commentedOn = FormatterMap.Standard.selectToGraphQL(modified.standard);
        // Remove join tables that are not present in GraphQL type, but present in Prisma
        modified = removeJoinTables(modified, commentFormatter().joinMapper);
        // Convert relationships
        modified = relationshipFormatter(modified, [
            ['reports', FormatterMap.Report.selectToGraphQL],
            ['starredBy', FormatterMap.User.selectToGraphQLUser],
        ]);
        return modified;
    },
});

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
const commenter = (format: CommentFormatterType, prisma: PrismaType) => ({
    async create(
        userId: string,
        input: CommentCreateInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<Comment>> {
        // Check for valid arguments
        commentCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Add comment
        let comment: any = await prisma.comment.create({
            data: {
                text: input.text,
                userId,
                [forMapper[input.createdFor]]: input.forId,
            },
            ...selectHelper(info, format.selectToDB)
        });
        // Return comment with fields calculated outside of the query
        return { ...format.selectToGraphQL(comment), isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: CommentUpdateInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<Comment>> {
        // Check for valid arguments
        commentUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Find comment
        let comment = await prisma.comment.findUnique({ where: { id: input.id } });
        if (!comment) throw new CustomError(CODE.NotFound, "Comment not found");
        // Update comment
        comment = await prisma.comment.update({
            where: { id: comment.id },
            data: {
                text: input.text,
            },
            ...selectHelper(info, format.selectToDB)
        });
        // Return comment with "isUpvoted" and "isStarred" fields. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { byId: userId, commentId: comment.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, commentId: comment.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.selectToGraphQL(comment), isUpvoted, isStarred };
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
            [authorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, comment.organizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.comment.delete({
            where: { id: comment.id },
        });
        return { success: true };
    },
    profanityCheck(data: CommentCreateInput | CommentUpdateInput): void {
        if (hasProfanity(data.text)) throw new CustomError(CODE.BannedWord);
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Comment;
    const format = commentFormatter();

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