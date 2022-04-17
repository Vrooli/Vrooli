import { CODE, MemberRole, standardCreate, standardTranslationCreate, standardTranslationUpdate, standardUpdate } from "@local/shared";
import { CustomError } from "../error";
import { PrismaType, RecursivePartial } from "types";
import { Standard, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSortBy, Count } from "../schema/types";
import { addCreatorField, addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, PartialInfo, relationshipToPrisma, removeCreatorField, removeJoinTablesHelper, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import _ from "lodash";
import { TranslationModel } from "./translation";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { tags: 'tag', starredBy: 'user' };
export const standardFormatter = (): FormatConverter<Standard> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Standard,
        'comments': GraphQLModelType.Comment,
        'creator': {
            'User': GraphQLModelType.User,
            'Organization': GraphQLModelType.Organization,
        },
        'reports': GraphQLModelType.Report,
        'routineInputs': GraphQLModelType.Routine,
        'routineOutputs': GraphQLModelType.Routine,
        'starredBy': GraphQLModelType.User,
        'tags': GraphQLModelType.Tag,
    },
    removeCalculatedFields: (partial) => {
        let { isUpvoted, isStarred, role, ...rest } = partial;
        return rest;
    },
    constructUnions: (data) => {
        let modified = addCreatorField(data);
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = removeCreatorField(partial);
        return modified;
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper)
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper)
    },
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialInfo,
    ): Promise<RecursivePartial<Standard>[]> {
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'standard')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, 'standard')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for role
        if (partial.role) {
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => Array.isArray(x.owner?.translations) && x.owner.translations.length > 0 && x.owner.translations[0].name)
                .map(x => x.owner.id)
                .filter(x => Boolean(x)) as string[];
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.owner?.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: x.creator?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        // Convert Prisma objects to GraphQL objects
        return objects as RecursivePartial<Standard>[];
    },
})

export const standardSearcher = (): Searcher<StandardSearchInput> => ({
    defaultSort: StandardSortBy.VotesDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [StandardSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [StandardSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [StandardSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [StandardSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [StandardSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [StandardSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [StandardSortBy.StarsAsc]: { stars: 'asc' },
            [StandardSortBy.StarsDesc]: { stars: 'desc' },
            [StandardSortBy.VotesAsc]: { score: 'asc' },
            [StandardSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                { name: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    },
    customQueries(input: StandardSearchInput): { [x: string]: any } {
        const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
        const minScoreQuery = input.minScore ? { score: { gte: input.minScore } } : {};
        const minStarsQuery = input.minStars ? { stars: { gte: input.minStars } } : {};
        const userIdQuery = input.userId ? { createdByUserId: input.userId } : {};
        const organizationIdQuery = input.organizationId ? { createdByOrganizationId: input.organizationId } : {};
        const projectIdQuery = input.projectId ? {
            OR: [
                { createdByUser: { projects: { some: { id: input.projectId } } } },
                { createdByOrganization: { projects: { some: { id: input.projectId } } } },
            ]
        } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const routineIdQuery = input.routineId ? {
            OR: [
                { routineInputs: { some: { routineId: input.routineId } } },
                { routineOutputs: { some: { routineId: input.routineId } } },
            ]
        } : {};
        const tagsQuery = input.tags ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {};
        return { ...languagesQuery, ...minScoreQuery, ...minStarsQuery, ...userIdQuery, ...organizationIdQuery, ...projectIdQuery, ...reportIdQuery, ...routineIdQuery, ...tagsQuery };
    },
})

export const standardVerifier = () => ({
    profanityCheck(data: StandardCreateInput | StandardUpdateInput): void {
        if (hasProfanity((data as any)?.name)) throw new CustomError(CODE.BannedWord);
        TranslationModel().profanityCheck(data);
    },
})

export const standardMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShapeAdd(userId: string | null, data: StandardCreateInput): Promise<any> {
        return {
            name: data.name,
            default: data.default,
            isFile: data.isFile,
            schema: data.schema,
            type: data.type,
            tags: await TagModel(prisma).relationshipBuilder(userId, data, GraphQLModelType.Standard),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, false),
        }
    },
    async toDBShapeUpdate(userId: string | null, data: StandardUpdateInput): Promise<any> {
        return {
            tags: await TagModel(prisma).relationshipBuilder(userId, data, GraphQLModelType.Standard),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, false),
        }
    },
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'standards',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes: string[] = [];
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as StandardCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: StandardUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create)) {
            formattedInput.create = formattedInput.create.map(async (data) => await this.toDBShapeAdd(userId, data as any));
        }
        if (Array.isArray(formattedInput.update)) {
            const updates = [];
            for (const update of formattedInput.update) {
                updates.push({
                    where: update.where,
                    data: await this.toDBShapeUpdate(userId, update.data as any),
                })
            }
            formattedInput.update = updates;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<StandardCreateInput, StandardUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => standardCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check for max standards created by user TODO
        }
        if (updateMany) {
            updateMany.forEach(input => standardUpdate.validateSync(input.data, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input.data));
        }
    },
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<StandardCreateInput, StandardUpdateInput>): Promise<CUDResult<Standard>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                let data = await this.toDBShapeAdd(userId, input);
                // Associate with either organization or user
                if (input.createdByOrganizationId) {
                    // Make sure the user is an admin of the organization
                    const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', input.createdByOrganizationId);
                    if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
                    data = {
                        ...data,
                        createdByOrganization: { connect: { id: input.createdByOrganizationId } },
                    };
                } else {
                    data = {
                        ...data,
                        createdByUser: { connect: { id: userId } },
                    };
                }
                // Create object
                const currCreated = await prisma.standard.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.standard.findUnique({
                    where: input.where,
                    select: {
                        id: true,
                        createdByUserId: true,
                        createdByOrganizationId: true,
                    }
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown);
                // Check if authorized to update
                if (!object) throw new CustomError(CODE.NotFound, 'Standard not found');
                if (object.createdByUserId && object.createdByUserId !== userId) throw new CustomError(CODE.Unauthorized);
                if (object.createdByOrganizationId) {
                    const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', object.createdByOrganizationId);
                    if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
                }
                // Update standard
                const currUpdated = await prisma.standard.update({
                    where: input.where,
                    data: await this.toDBShapeUpdate(userId, input.data),
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            const objects = await prisma.standard.findMany({
                where: { id: { in: deleteMany } },
                select: {
                    id: true,
                    createdByUserId: true,
                    createdByOrganizationId: true,
                }
            });
            // Filter out objects that match the user's Id, since we know those are authorized
            const objectsToCheck = objects.filter(object => object.createdByUserId !== userId);
            if (objectsToCheck.length > 0) {
                for (const check of objectsToCheck) {
                    // Check if user is authorized to delete
                    if (!check.createdByOrganizationId) throw new CustomError(CODE.Unauthorized, 'Not authorized to delete');
                    const [authorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', check.createdByOrganizationId);
                    if (!authorized) throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.');
                }
            }
            // Delete
            deleted = await prisma.standard.deleteMany({
                where: { id: { in: deleteMany } },
            });
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

export function StandardModel(prisma: PrismaType) {
    const prismaObject = prisma.standard
    const format = standardFormatter();
    const search = standardSearcher();
    const verify = standardVerifier();
    const mutate = standardMutater(prisma, verify);

    return {
        prisma,
        prismaObject,
        format,
        ...format,
        ...search,
        ...verify,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================