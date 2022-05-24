import { CODE, MemberRole, standardsCreate, standardsUpdate, standardTranslationCreate, standardTranslationUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { PrismaType, RecursivePartial } from "types";
import { Standard, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSortBy, Count } from "../../schema/types";
import { addCreatorField, addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, PartialInfo, relationshipToPrisma, removeCreatorField, removeJoinTablesHelper, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { validateProfanity } from "../../utils/censor";
import { OrganizationModel } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import _ from "lodash";
import { TranslationModel } from "./translation";
import { genErrorCode } from "../../logger";
import { ViewModel } from "./view";
import { randomString } from "../../auth/walletAuth";
import { sortify } from "../../utils/objectTools";

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
                ? await StarModel(prisma).getIsStarreds(userId, ids, GraphQLModelType.Standard)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, GraphQLModelType.Standard)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for isViewed
        if (partial.isViewed) {
            const isViewedArray = userId
                ? await ViewModel(prisma).getIsVieweds(userId, ids, GraphQLModelType.Standard)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isViewed: isViewedArray[i] }));
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
        return {
            ...(input.languages ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minScore ? { score: { gte: input.minScore } } : {}),
            ...(input.minStars ? { stars: { gte: input.minStars } } : {}),
            ...(input.minViews ? { views: { gte: input.minViews } } : {}),
            ...(input.userId ? { createdByUserId: input.userId } : {}),
            ...(input.organizationId ? { createdByOrganizationId: input.organizationId } : {}),
            ...(input.projectId ? {
                OR: [
                    { createdByUser: { projects: { some: { id: input.projectId } } } },
                    { createdByOrganization: { projects: { some: { id: input.projectId } } } },
                ]
            } : {}),
            ...(input.reportId ? { reports: { some: { id: input.reportId } } } : {}),
            ...(input.routineId ? {
                OR: [
                    { routineInputs: { some: { routineId: input.routineId } } },
                    { routineOutputs: { some: { routineId: input.routineId } } },
                ]
            } : {}),
            ...(input.tags ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        }
    },
})

export const standardVerifier = () => ({
    profanityCheck(data: (StandardCreateInput | StandardUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.name));
        TranslationModel().profanityCheck(data);
    },
})

export const standardQuerier = (prisma: PrismaType) => ({
    /**
     * Checks if a standard exists that has an identical shape to the given standard. 
     * This is useful to preventing duplicate standards from being created.
     * @param data StandardCreateData to check
     * @returns ID of matching standard, or null if none found
     */
    async findMatchingStandard(data: StandardCreateInput): Promise<string | null> {
        // Sort all JSON properties that are part of the comparison
        const props = sortify(data.props);
        const yup = data.yup ? sortify(data.yup) : null;
        // name ,default
        // Find all standards that match the given standard
        const standards = await prisma.standard.findMany({
            where: {
                default: data.default ?? null,
                props: props,
                yup: yup,
            }
        });
        // If any standards match (should only ever be 0 or 1, but you never know) return the first one
        if (standards.length > 0) {
            return standards[0].id;
        }
        // If no standards match, then data is unique. Return null
        return null;
    },
    /**
     * Generates a valid name for a standard.
     * Standards must have a unique name/version pair per user/organization
     * @param userId The user's ID
     * @param data The standard create data
     * @returns A valid name for the standard
     */
    async generateName(userId: string, data: StandardCreateInput): Promise<string> {
        // Created by query
        const id = data.createdByOrganizationId ?? data.createdByUserId ?? userId
        const createdBy = { [`createdBy${data.createdByOrganizationId ? GraphQLModelType.Organization : GraphQLModelType.User}Id`]: id };
        // Calculate optional standard name
        const name = data.name ? data.name : `${data.type} ${randomString(5)}`;
        // Loop until a unique name is found, or a max of 20 tries
        let success = false;
        let i = 0;
        while (!success && i < 20) {
            // Check for case-insensitive duplicate
            const existing = await prisma.standard.findMany({
                where: {
                    ...createdBy,
                    name: {
                        contains: (i === 0 ? name : `${name}${i}`).toLowerCase(),
                        mode: 'insensitive',
                    },
                    version: data.version ?? undefined,
                }
            });
            if (existing.length > 0) i++;
            else success = true;
        }
        return i === 0 ? name : `${name}${i}`;
    }
})

export const standardMutater = (prisma: PrismaType, verifier: ReturnType<typeof standardVerifier>) => ({
    async toDBShapeAdd(userId: string | null, data: StandardCreateInput): Promise<any> {
        return {
            name: await standardQuerier(prisma).generateName(userId ?? '', data),
            default: data.default,
            type: data.type,
            props: sortify(data.props),
            yup: data.yup ? sortify(data.yup) : undefined,
            tags: await TagModel(prisma).relationshipBuilder(userId, data, GraphQLModelType.Standard),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, false),
            version: data.version ?? '1.0.0',
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
            // For all creates that are exact matches to existing standards, make them connects instead
            const newConnectIds: string[] = [];
            for (const create of formattedInput.create) {
                const existingId = await standardQuerier(prisma).findMatchingStandard(create as any)
                if (existingId) {
                    newConnectIds.push(existingId);
                }
            }
            // Add newConnectIds to formattedInput.connect
            if (newConnectIds.length > 0) {
                formattedInput.connect = formattedInput.connect ?? [];
                formattedInput.connect = [...formattedInput.connect, ...newConnectIds.map(id => ({ id }))];
                // Remove newConnectIds from formattedInput.create
                formattedInput.create = formattedInput.create.filter(create => !newConnectIds.includes(create.id));
            }
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
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId) 
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0103') });
        // Collect organizationIds from each object, and check if the user is an admin/owner of every organization
        const organizationIds: (string | null | undefined)[] = [];
        if (createMany) {
            standardsCreate.validateSync(createMany, { abortEarly: false });
            verifier.profanityCheck(createMany);
            // Add createdByOrganizationIds to organizationIds array, if they are set
            organizationIds.push(...createMany.map(input => input.createdByOrganizationId).filter(id => id));
            // Check for max standards created by user TODO
        }
        if (updateMany) {
            standardsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            verifier.profanityCheck(updateMany.map(u => u.data));
            // Add existing organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.standard.findMany({
                where: { id: { in: updateMany.map(input => input.where.id) } },
                select: { id: true, createdByUserId: true, createdByOrganizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.createdByUserId !== userId).map(object => object.createdByOrganizationId));
        }
        if (deleteMany) {
            // Add organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.standard.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, createdByUserId: true, createdByOrganizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.createdByUserId !== userId).map(object => object.createdByOrganizationId));
        }
        // Find admin/owner member data for every organization
        const memberData = await OrganizationModel(prisma).isOwnerOrAdmin(userId, organizationIds);
        // If any member data is undefined, the user is not authorized to delete one or more objects
        if (memberData.some(member => !member))
            throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.', { code: genErrorCode('0095') })
    },
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<StandardCreateInput, StandardUpdateInput>): Promise<CUDResult<Standard>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                // TODO check for exact matches, and don't add
                let data = await this.toDBShapeAdd(userId, input);
                // Associate with either organization or user
                if (input.createdByOrganizationId) {
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
                if (!object) 
                    throw new CustomError(CODE.ErrorUnknown, 'Standard not found', { code: genErrorCode('0105') });
                // Check if authorized to update
                if (!object) 
                    throw new CustomError(CODE.NotFound, 'Standard not found', { code: genErrorCode('0106') });
                if (object.createdByUserId && object.createdByUserId !== userId) 
                    throw new CustomError(CODE.Unauthorized, 'Not authorized to update standard', { code: genErrorCode('0107') });
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
    const query = standardQuerier(prisma);
    const mutate = standardMutater(prisma, verify);

    return {
        prisma,
        prismaObject,
        format,
        ...format,
        ...search,
        ...verify,
        ...query,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================