import { standardsCreate, standardsUpdate, standardTranslationCreate, standardTranslationUpdate } from "@shared/validation";
import { CODE, DeleteOneType } from "@shared/consts";
import { omit } from '@shared/utils'; 
import { CustomError } from "../../error";
import { PrismaType, RecursivePartial } from "../../types";
import { Standard, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSortBy, Count, StandardPermission } from "../../schema/types";
import { addCountFieldsHelper, addCreatorField, addJoinTablesHelper, addSupplementalFieldsHelper, CUDInput, CUDResult, deleteOneHelper, FormatConverter, getSearchStringQueryHelper, modelToGraphQL, onlyValidIds, PartialGraphQLInfo, Permissioner, relationshipToPrisma, removeCountFieldsHelper, removeCreatorField, removeJoinTablesHelper, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { validateProfanity } from "../../utils/censor";
import { OrganizationModel } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { TranslationModel } from "./translation";
import { genErrorCode } from "../../logger";
import { ViewModel } from "./view";
import { randomString } from "../../auth/walletAuth";
import { sortify } from "../../utils/objectTools";
import { ResourceListModel } from "./resourceList";
import { v4 as uuid } from 'uuid';

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { tags: 'tag', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
const supplementalFields = ['isUpvoted', 'isStarred', 'isViewed', 'permissionsStandard'];
export const standardFormatter = (): FormatConverter<Standard, StandardPermission> => ({
    relationshipMap: {
        '__typename': 'Standard',
        'comments': 'Comment',
        'creator': {
            'User': 'User',
            'Organization': 'Organization',
        },
        'reports': 'Report',
        'resourceLists': 'ResourceList',
        'routineInputs': 'Routine',
        'routineOutputs': 'Routine',
        'starredBy': 'User',
        'tags': 'Tag',
    },
    constructUnions: (data) => {
        let modified = addCreatorField(data);
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = removeCreatorField(partial);
        return modified;
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    removeSupplementalFields: (partial) => omit(partial, supplementalFields),
    async addSupplementalFields({ objects, partial, permissions, prisma, userId }): Promise<RecursivePartial<Standard>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Standard')],
                ['isUpvoted', async (ids) => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Standard')],
                ['isViewed', async (ids) => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Standard')],
                ['permissionsStandard', async () => await StandardModel.permissions(prisma).get({ objects, permissions, userId })],
            ]
        });
    }
})

export const standardPermissioner = (prisma: PrismaType): Permissioner<StandardPermission, StandardSearchInput> => ({
    async get({
        objects,
        permissions,
        userId,
    }) {
        // Initialize result with ID
        const result = objects.map((o) => ({
            canComment: true,
            canDelete: false,
            canEdit: false,
            canReport: true,
            canStar: true,
            canView: true,
            canVote: true,
        }));
        if (!userId) return result;
        const ids = objects.map(x => x.id);
        let creatorData: { 
            id: string, 
            user?: { id: string } | null | undefined, 
            organization?: { id: string } | null | undefined
        }[] = [];
        // If some creator data missing, query for creator data.
        if (onlyValidIds(objects.map(x => x.creator)).length < objects.length) {
            creatorData = await prisma.standard.findMany({
                where: { id: { in: ids } },
                select: {
                    id: true,
                    createdByUser: { select: { id: true } },
                    createdByOrganization: { select: { id: true } },
                },
            });
        } else {
            creatorData = objects.map((x) => {
                const isOrg = Boolean(Array.isArray(x.creator?.translations) && x.creator.translations.length > 0 && x.creator.translations[0].name);
                return ({
                    id: x.id,
                    user: isOrg ? null : x.creator,
                    organization: isOrg ? x.creator : null,
                });
            });
        }
        // Find permissions for every organization
        const organizationIds = onlyValidIds(creatorData.map(x => x.organization?.id));
        const orgPermissions = await OrganizationModel.permissions(prisma).get({ 
            objects: organizationIds.map(x => ({ id: x })),
            userId 
        });
        // Find which objects have ownership permissions
        for (let i = 0; i < objects.length; i++) {
            const unformatted = creatorData.find(y => y.id === objects[i].id);
            if (!unformatted) continue;
            // Check if user owns object directly, or through organization
            if (unformatted.user?.id !== userId) {
                const orgIdIndex = organizationIds.findIndex(id => id === unformatted?.organization?.id);
                if (orgIdIndex < 0) continue;
                if (!orgPermissions[orgIdIndex].canEdit) continue;
            }
            // Set creator permissions
            result[i].canDelete = true;
            result[i].canEdit = true;
            result[i].canView = true;
        }
        // TODO isPrivate view check
        // TODO check relationships for permissions
        return result;
    },
    async canSearch({
        input,
        userId
    }) {
        //TODO
        return 'full';
    },
    ownershipQuery: (userId) => ({
        OR: [
            { createdByOrganization: { roles: { some: { assignees: { some: { user: { id: userId } } } } } } },
            { createdByUser: { id: userId } }
        ]
    }),
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
        return getSearchStringQueryHelper({ searchString,
            resolver: ({ insensitive }) => ({ 
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { name: { ...insensitive } },
                    { tags: { some: { tag: { tag: { ...insensitive } } } } },
                ]
            })
        })
    },
    customQueries(input: StandardSearchInput): { [x: string]: any } {
        return {
            /**
             * isInternal routines should never appear in the query, since they are 
             * only meant for a single input/output
             */
            isInternal: false,
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            ...(input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            ...(input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            ...(input.userId !== undefined ? { createdByUserId: input.userId } : {}),
            ...(input.organizationId !== undefined ? { createdByOrganizationId: input.organizationId } : {}),
            ...(input.projectId !== undefined ? {
                OR: [
                    { createdByUser: { projects: { some: { id: input.projectId } } } },
                    { createdByOrganization: { projects: { some: { id: input.projectId } } } },
                ]
            } : {}),
            ...(input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            ...(input.routineId !== undefined ? {
                OR: [
                    { routineInputs: { some: { routineId: input.routineId } } },
                    { routineOutputs: { some: { routineId: input.routineId } } },
                ]
            } : {}),
            ...(input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
            ...(!!input.type ? { type: { contains: input.type.trim(), mode: 'insensitive' } } : {}),
        }
    },
})

export const standardVerifier = () => ({
    profanityCheck(data: (StandardCreateInput | StandardUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.name));
        TranslationModel.profanityCheck(data);
    },
})

export const standardQuerier = (prisma: PrismaType) => ({
    /**
     * Checks if a standard exists that has an identical shape to the given standard. 
     * This is useful to preventing duplicate standards from being created.
     * @param data StandardCreateData to check
     * @param userId The ID of the user creating the standard
     * @param uniqueToCreator Whether to check if the standard is unique to the user/organization 
     * @param isInternal Used to determine if the standard should show up in search results
     * @returns data of matching standard, or null if no match
     */
    async findMatchingStandardShape(
        data: StandardCreateInput,
        userId: string,
        uniqueToCreator: boolean,
        isInternal: boolean | null,
    ): Promise<{ [x: string]: any } | null> {
        // Sort all JSON properties that are part of the comparison
        const props = sortify(data.props);
        const yup = data.yup ? sortify(data.yup) : null;
        // Find all standards that match the given standard
        const standards = await prisma.standard.findMany({
            where: {
                isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
                default: data.default ?? null,
                props: props,
                yup: yup,
                createdByUserId: (uniqueToCreator && !data.createdByOrganizationId) ? userId : undefined,
                createdByOrganizationId: (uniqueToCreator && data.createdByOrganizationId) ? data.createdByOrganizationId : undefined,
            }
        });
        // If any standards match (should only ever be 0 or 1, but you never know) return the first one
        if (standards.length > 0) {
            return standards[0];
        }
        // If no standards match, then data is unique. Return null
        return null;
    },
    /**
     * Checks if a standard exists that has the same createdByUserId, 
     * createdByOrganizationId, name, and version
     * @param data StandardCreateData to check
     * @param userId The ID of the user creating the standard
     * @returns data of matching standard, or null if no match
     */
    async findMatchingStandardName(
        data: StandardCreateInput & { name: string, version: string },
        userId: string
    ): Promise<{ [x: string]: any } | null> {
        // Find all standards that match the given standard
        const standards = await prisma.standard.findMany({
            where: {
                name: data.name,
                version: data.version,
                createdByUserId: !data.createdByOrganizationId ? userId : undefined,
                createdByOrganizationId: data.createdByOrganizationId ? data.createdByOrganizationId : undefined,
            }
        });
        // If any standards match (should only ever be 0 or 1, but you never know) return the first one
        if (standards.length > 0) {
            return standards[0];
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
        const createdBy = { [`createdBy${data.createdByOrganizationId ? 'Organization' : 'User'}Id`]: id };
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

export const standardMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string | null, data: StandardCreateInput): Promise<any> {
        let translations = TranslationModel.relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, true)
        if (translations?.jsonVariable) {
            translations.jsonVariable = sortify(translations.jsonVariable);
        }
        return {
            id: data.id,
            isInternal: data.isInternal ?? false,
            name: await standardQuerier(prisma).generateName(userId ?? '', data),
            default: data.default,
            type: data.type,
            props: sortify(data.props),
            yup: data.yup ? sortify(data.yup) : undefined,
            resourceLists: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, true),
            tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Standard'),
            translations,
            version: data.version ?? '1.0.0',
            versionGroupId: uuid(),
        }
    },
    async toDBShapeUpdate(userId: string | null, data: StandardUpdateInput): Promise<any> {
        let translations = TranslationModel.relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, false)
        if (translations?.jsonVariable) {
            translations.jsonVariable = sortify(translations.jsonVariable);
        }
        return {
            resourceLists: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Standard'),
            translations,
        }
    },
    /**
     * Add, update, or remove a one-to-one standard relationship. 
     * Due to some unknown Prisma bug, it won't let us create/update a standard directly
     * in the main mutation query like most other relationship builders. Instead, we 
     * must do this separately, and return the standard's ID.
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<string | null> {
        // Convert input to Prisma shape
        const fieldExcludes: string[] = [];
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName: 'standard', isAdd, fieldExcludes })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as StandardCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: StandardUpdateInput }[],
            deleteMany: deleteMany?.map((d: any) => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            let create: any;
            // If standard is internal, check if the shape exists in the database
            if (formattedInput.create[0].isInternal) {
                const existingStandard = await standardQuerier(prisma).findMatchingStandardShape(formattedInput.create[0], userId ?? '', false, true);
                // If standard found, connect instead of create
                if (existingStandard) {
                    return existingStandard.id;
                }
            }
            // Otherwise, perform two unique checks
            // 1. Check if standard with same createdByUserId, createdByOrganizationId, name, and version already exists with the same creator
            // 2. Check if standard of same shape already exists with the same creator
            // If the first check returns a standard, throw error
            // If the second check returns a standard, then connect the existing standard.
            else {
                // First call createData helper function, so we can use the generated name
                create = await this.toDBShapeAdd(userId, formattedInput.create[0]);
                const check1 = await standardQuerier(prisma).findMatchingStandardName(create, userId ?? '');
                if (check1) {
                    throw new CustomError(CODE.StandardDuplicateName, 'Standard with this name/version pair already exists.', { code: genErrorCode('0240') });
                }
                const check2 = await standardQuerier(prisma).findMatchingStandardShape(create, userId ?? '', true, false)
                if (check2) {
                    return check2.id;
                }
            }
            // Shape create data
            if (!create) create = await this.toDBShapeAdd(userId, formattedInput.create[0]);
            // Create standard
            const standard = await prisma.standard.create({
                data: create,
            })
            return standard?.id ?? null;
        }
        if (Array.isArray(formattedInput.connect) && formattedInput.connect.length > 0) {
            return formattedInput.connect[0].id;
        }
        if (Array.isArray(formattedInput.disconnect) && formattedInput.disconnect.length > 0) {
            return null;
        }
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            const update = await this.toDBShapeUpdate(userId, formattedInput.update[0]);
            // Update standard
            const standard = await prisma.standard.update({
                where: { id: update.id },
                data: update,
            })
            return standard.id;
        }
        if (Array.isArray(formattedInput.delete) && formattedInput.delete.length > 0) {
            const deleteId = formattedInput.delete[0].id;
            // If standard is internal, disconnect instead
            if (input.isInternal) return null;
            // Delete standard
            await deleteOneHelper({
                input: { id: deleteId, objectType: DeleteOneType.Standard },
                model: StandardModel,
                prisma,
                userId,
            })
            return deleteId;
        }
        return null;
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
            standardVerifier().profanityCheck(createMany);
            // Add createdByOrganizationIds to organizationIds array, if they are set
            organizationIds.push(...onlyValidIds(createMany.map(input => input.createdByOrganizationId)));
            // Check for max standards created by user TODO
        }
        if (updateMany) {
            standardsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            standardVerifier().profanityCheck(updateMany.map(u => u.data));
            // Add existing organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.standard.findMany({
                where: { id: { in: updateMany.map(input => input.where.id) } },
                select: { id: true, createdByUserId: true, createdByOrganizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.createdByUserId !== userId).map(object => object.createdByOrganizationId));
        }
        if (deleteMany) {
            const objects = await prisma.standard.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, createdByUserId: true, createdByOrganizationId: true },
            });
            // Split objects by userId and organizationId
            const userIds = objects.filter(object => Boolean(object.createdByUserId)).map(object => object.createdByUserId);
            if (userIds.some(id => id !== userId))
                throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.', { code: genErrorCode('0244') })
            // Add to organizationIds array, to check ownership status
            organizationIds.push(...objects.filter(object => !userId.includes(object.createdByOrganizationId ?? '')).map(object => object.createdByOrganizationId));
        }
        // Find role for every organization
        const roles = await OrganizationModel.query(prisma).hasRole(userId, organizationIds);
        // If any role is undefined, the user is not authorized to delete one or more objects
        if (roles.some(role => !role))
            throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.', { code: genErrorCode('0251') })
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<StandardCreateInput, StandardUpdateInput>): Promise<CUDResult<Standard>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        const select = selectHelper(partialInfo);
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            let data: any;
            // Loop through each create input
            for (const input of createMany) {
                // If standard is internal, check if the shape exists in the database
                if (input.isInternal) {
                    const existingStandard = await standardQuerier(prisma).findMatchingStandardShape(input, userId ?? '', false, true);
                    // If standard found, pretend it was created
                    if (existingStandard) {
                        // Find full data
                        const existingData = await prisma.standard.findUnique({ where: { id: existingStandard.id }, ...selectHelper(partialInfo) });
                        // Convert to GraphQL
                        const converted = modelToGraphQL(existingData as any, partialInfo);
                        // Add to created array
                        created = created ? [...created, converted] : [converted];
                        continue;
                    }
                }
                // Otherwise, perform two unique checks
                // 1. Check if standard with same createdByUserId, createdByOrganizationId, name, and version already exists with the same creator
                // 2. Check if standard of same shape already exists with the same creator
                // If any checks return an existing standard, throw error
                else {
                    // First call createData helper function, so we can use the generated name
                    data = await this.toDBShapeAdd(userId, input);
                    const check1 = await standardQuerier(prisma).findMatchingStandardName(data, userId ?? '');
                    if (check1) {
                        throw new CustomError(CODE.StandardDuplicateName, 'Standard with this name/version pair already exists.', { code: genErrorCode('0238') });
                    }
                    const check2 = await standardQuerier(prisma).findMatchingStandardShape(data, userId ?? '', true, false)
                    if (check2) {
                        throw new CustomError(CODE.StandardDuplicateShape, 'Standard with this shape already exists.', { code: genErrorCode('0239') });
                    }
                }
                // If not called, create data
                if (!data) data = await this.toDBShapeAdd(userId, input);
                // If not internal, associate with either organization or user
                if (!input.isInternal) {
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
                }
                // Create object
                const currCreated = await prisma.standard.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
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
                    ...select
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            // While standards can be deleted, we must be careful not to delete standards that are referenced by other objects
            // For example, a standard used by routines will be anonimized instead of deleted.
            for (const id of deleteMany) {
                // Check if standard is used by any inputs/outputs
                const standard = await prisma.standard.findUnique({
                    where: { id },
                    select: {
                        _count: {
                            select: {
                                routineInputs: true,
                                routineOutputs: true,
                            }
                        }
                    }
                })
                // If standard not found, throw error
                if (!standard) {
                    throw new CustomError(CODE.NotFound, 'Standard not found', { code: genErrorCode('0241') });
                }
                // If standard is being used, anonymize
                if (standard._count.routineInputs || standard._count.routineOutputs) {
                    await prisma.standard.update({
                        where: { id },
                        data: {
                            createdByOrganizationId: null,
                            createdByUserId: null,
                        }
                    })
                }
                // Otherwise, delete (unless it is internal)
                else {
                    await prisma.standard.deleteMany({
                        where: {
                            id,
                            isInternal: false
                        }
                    });
                    deleted.count++;
                }
            }
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

export const StandardModel = ({
    prismaObject: (prisma: PrismaType) => prisma.standard,
    format: standardFormatter(),
    mutate: standardMutater,
    permissions: standardPermissioner,
    query: standardQuerier,
    search: standardSearcher(),
    verify: standardVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================