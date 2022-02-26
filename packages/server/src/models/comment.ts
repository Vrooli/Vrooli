import { CODE, commentCreate, CommentFor, commentUpdate } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentCreateInput, CommentUpdateInput, Count } from "../schema/types";
import { PrismaType } from "types";
import { addCreatorField, addJoinTablesHelper, addSupplementalFields, CUDInput, CUDResult, deconstructUnion, FormatConverter, removeCreatorField, removeJoinTablesHelper, selectHelper, modelToGraphQL, ValidateMutationsInput } from "./base";
import { hasProfanity } from "../utils/censor";
import { organizationVerifier } from "./organization";
import { routineDBFields } from "./routine";
import _ from "lodash";
import { projectDBFields } from "./project";
import { standardDBFields } from "./standard";
import { MemberRole } from "@prisma/client";

export const commentDBFields = ['id', 'text', 'created_at', 'updated_at', 'userId', 'organizationId', 'projectId', 'routineId', 'standardId', 'score', 'stars']

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { starredBy: 'user' };
export const commentFormatter = (): FormatConverter<Comment> => ({
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
        return modified;
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
})

export const commentVerifier = () => ({
    profanityCheck(data: CommentCreateInput | CommentUpdateInput): void {
        if (hasProfanity(data.text)) throw new CustomError(CODE.BannedWord);
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
export const commentMutater = (prisma: PrismaType, verifier: any) => ({
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<CommentCreateInput, CommentUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => commentCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // TODO check limits on comments to prevent spam
        }
        if (updateMany) {
            updateMany.forEach(input => commentUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input));
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
    async cud({ info, userId, createMany, updateMany, deleteMany }: CUDInput<CommentCreateInput, CommentUpdateInput>): Promise<CUDResult<Comment>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Create object
                const currCreated = await prisma.comment.create({
                    data: {
                        text: input.text,
                        userId,
                        [forMapper[input.createdFor]]: input.forId,
                    },
                    ...selectHelper(info)
                })
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, info);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find comment
                let comment = await prisma.comment.findUnique({ where: { id: input.id } });
                if (!comment) throw new CustomError(CODE.NotFound, "Comment not found");
                // Update comment
                const currUpdated = await prisma.comment.update({
                    where: { id: comment.id },
                    data: {
                        text: input.text,
                    },
                    ...selectHelper(info)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, info);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.comment.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        // Format and add supplemental/calculated fields
        const createdLength = created.length;
        const supplemental = await addSupplementalFields(prisma, userId, [...created, ...updated], info);
        created = supplemental.slice(0, createdLength);
        updated = supplemental.slice(createdLength);
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
    const verifier = commentVerifier();
    const mutater = commentMutater(prisma, verifier);

    return {
        prismaObject,
        ...format,
        ...verifier,
        ...mutater,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================