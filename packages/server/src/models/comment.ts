import { CODE, commentCreate, CommentSortBy, commentTranslationCreate, commentTranslationUpdate, commentUpdate } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentCreateInput, CommentFor, CommentSearchInput, CommentUpdateInput, Count } from "../schema/types";
import { PrismaType } from "types";
import { addCreatorField, addJoinTablesHelper, CUDInput, CUDResult, deconstructUnion, FormatConverter, removeCreatorField, removeJoinTablesHelper, selectHelper, modelToGraphQL, ValidateMutationsInput, Searcher, GraphQLModelType } from "./base";
import { hasProfanity } from "../utils/censor";
import { organizationVerifier } from "./organization";
import pkg from '@prisma/client';
import { TranslationModel } from "./translation";
const { MemberRole } = pkg;

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user' };
export const commentFormatter = (): FormatConverter<Comment> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Comment,
        'user': GraphQLModelType.User,
        'organization': GraphQLModelType.Organization,
        'project': GraphQLModelType.Project,
        'routine': GraphQLModelType.Routine,
        'standard': GraphQLModelType.Standard,
        'reports': GraphQLModelType.Report,
        'starredBy': GraphQLModelType.User,
        'votes': GraphQLModelType.Vote,
    },
    removeCalculatedFields: (partial) => {
        let { isUpvoted, isStarred, ...rest } = partial;
        return rest;
    },
    constructUnions: (data) => {
        let { project, routine, standard, ...modified } = addCreatorField(data);
        if (project) modified.commentedOn = modified.project;
        else if (routine) modified.commentedOn = modified.routine;
        else if (standard) modified.commentedOn = modified.standard;
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = removeCreatorField(partial);
        modified = deconstructUnion(partial, 'commentedOn', [
            [GraphQLModelType.Project, 'project'],
            [GraphQLModelType.Routine, 'routine'],
            [GraphQLModelType.Standard, 'standard'],
        ]);
        return modified;
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
})

export const commentSearcher = (): Searcher<CommentSearchInput> => ({
    defaultSort: CommentSortBy.VotesDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [CommentSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [CommentSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [CommentSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [CommentSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [CommentSortBy.StarsAsc]: { stars: 'asc' },
            [CommentSortBy.StarsDesc]: { stars: 'desc' },
            [CommentSortBy.VotesAsc]: { score: 'asc' },
            [CommentSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({ translations: { some: { language: languages ? { in: languages } : undefined, text: {...insensitive} } } });
    },
    customQueries(input: CommentSearchInput): { [x: string]: any } {
        const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
        const minScoreQuery = input.minScore ? { score: { gte: input.minScore } } : {};
        const minStarsQuery = input.minStars ? { stars: { gte: input.minStars } } : {};
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const projectIdQuery = input.projectId ? { projectId: input.projectId } : undefined;
        const routineIdQuery = input.routineId ? { routineId: input.routineId } : undefined;
        const standardIdQuery = input.standardId ? { standardId: input.standardId } : undefined;
        return { ...languagesQuery, ...minScoreQuery, ...minStarsQuery, ...userIdQuery, ...organizationIdQuery, ...projectIdQuery, ...routineIdQuery, ...standardIdQuery };
    },
})

const forMapper = {
    [CommentFor.Project]: 'projectId',
    [CommentFor.Routine]: 'routineId',
    [CommentFor.Standard]: 'standardId',
}

/**
 * Handles authorized creates, updates, and deletes
 */
export const commentMutater = (prisma: PrismaType) => ({
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<CommentCreateInput, CommentUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => commentCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
            // TODO check limits on comments to prevent spam
        }
        if (updateMany) {
            updateMany.forEach(input => commentUpdate.validateSync(input.data, { abortEarly: false }));
            updateMany.forEach(input => TranslationModel().profanityCheck(input.data));
        }
        if (deleteMany) {
            // Check that user created each comment
            const comments = await prisma.comment.findMany({
                where: { id: { in: deleteMany } },
                select: {
                    id: true,
                    userId: true,
                    organizationId: true,
                }
            })
            // Filter out comments that user created
            const notCreatedByThisUser = comments.filter(c => c.userId !== userId);
            // If any comments not created by this user have a null organizationId, throw error
            if (notCreatedByThisUser.some(c => c.organizationId === null)) throw new CustomError(CODE.Unauthorized);
            // Of the remaining comments, check that user is an admin of the organization
            const organizationIds = notCreatedByThisUser.map(c => c.organizationId).filter(id => id !== null) as string[];
            const roles = userId
                ? await organizationVerifier(prisma).getRoles(userId, organizationIds)
                : Array(deleteMany.length).fill(null);
            if (roles.some((role: any) => role !== MemberRole.Owner && role !== MemberRole.Admin)) throw new CustomError(CODE.Unauthorized);
        }
    },
    /**
     * Performs adds, updates, and deletes of organizations. First validates that every action is allowed.
     */
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<CommentCreateInput, CommentUpdateInput>): Promise<CUDResult<Comment>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Create object
                const currCreated = await prisma.comment.create({
                    data: {
                        translations: TranslationModel().relationshipBuilder(userId, input, { create: commentTranslationCreate, update: commentTranslationUpdate }, false),
                        userId,
                        [forMapper[input.createdFor]]: input.forId,
                    },
                    ...selectHelper(partial)
                })
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find comment
                let comment = await prisma.comment.findUnique({ where: input.where });
                if (!comment) throw new CustomError(CODE.NotFound, "Comment not found");
                // Update comment
                const currUpdated = await prisma.comment.update({
                    where: input.where,
                    data: {
                        translations: TranslationModel().relationshipBuilder(userId, input.data, { create: commentTranslationCreate, update: commentTranslationUpdate }, false),
                    },
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.comment.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma: PrismaType) {
    const prismaObject = prisma.comment;
    const format = commentFormatter();
    const search = commentSearcher();
    const mutater = commentMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...mutater,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================