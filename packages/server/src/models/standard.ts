import { standardTranslationCreate, standardTranslationUpdate } from "@shared/validation";
import { CODE, DeleteOneType, StandardSortBy } from "@shared/consts";
import { omit } from '@shared/utils';
import { addCountFieldsHelper, addJoinTablesHelper, addSupplementalFieldsHelper, combineQueries, getSearchStringQueryHelper, modelToGraphQL, onlyValidIds, removeCountFieldsHelper, removeJoinTablesHelper, selectHelper, visibilityBuilder } from "./builder";
import { organizationQuerier } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { TranslationModel } from "./translation";
import { ViewModel } from "./view";
import { ResourceListModel } from "./resourceList";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, Searcher, Validator } from "./types";
import { randomString } from "../auth/walletAuth";
import { CustomError, genErrorCode } from "../events";
import { Standard, StandardPermission, StandardSearchInput, StandardCreateInput, StandardUpdateInput, Count } from "../schema/types";
import { RecursivePartial, PrismaType } from "../types";
import { sortify } from "../utils/objectTools";
import { deleteOneHelper } from "./actions";
import { Prisma } from "@prisma/client";
import { AnyAaaaRecord } from "dns";
import { getPermissions } from "./utils";

const joinMapper = { tags: 'tag', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
const supplementalFields = ['isUpvoted', 'isStarred', 'isViewed', 'permissionsStandard', 'versions'];
export const standardFormatter = (): FormatConverter<Standard, StandardPermission> => ({
    relationshipMap: {
        '__typename': 'Standard',
        'comments': 'Comment',
        'creator': {
            'root': {
                'User': 'User',
                'Organization': 'Organization',
            }
        },
        'reports': 'Report',
        'resourceLists': 'ResourceList',
        'routineInputs': 'Routine',
        'routineOutputs': 'Routine',
        'starredBy': 'User',
        'tags': 'Tag',
    },
    rootFields: ['hasCompleteVersion', 'isDeleted', 'isInternal', 'isPrivate', 'name', 'votes', 'stars', 'views', 'permissions'],
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    removeSupplementalFields: (partial) => omit(partial, supplementalFields),
    async addSupplementalFields({ objects, partial, prisma, userId }): Promise<RecursivePartial<Standard>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Standard')],
                ['isUpvoted', async (ids) => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Standard')],
                ['isViewed', async (ids) => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Standard')],
                ['permissionsStandard', async (ids) => await getPermissions({ objectType: 'Standard', ids, prisma, userId })],
                ['versions', async (ids) => {
                    const groupData = await prisma.standard.findMany({
                        where: {
                            id: { in: ids },
                        },
                        select: {
                            versions: {
                                select: { id: true, versionLabel: true }
                            }
                        }
                    });
                    return groupData.map(g => g.versions);
                }],
            ]
        });
    }
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
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { name: { ...insensitive } },
                    { tags: { some: { tag: { tag: { ...insensitive } } } } },
                ]
            })
        })
    },
    customQueries(input: StandardSearchInput, userId: string | null | undefined): { [x: string]: any } {
        return combineQueries([
            /**
             * isInternal routines should never appear in the query, since they are 
             * only meant for a single input/output
             */
            { isInternal: false },
            visibilityBuilder({ model: StandardModel, userId, visibility: input.visibility }),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            (input.userId !== undefined ? { createdByUserId: input.userId } : {}),
            (input.organizationId !== undefined ? { createdByOrganizationId: input.organizationId } : {}),
            (input.projectId !== undefined ? {
                OR: [
                    { createdByUser: { projects: { some: { id: input.projectId } } } },
                    { createdByOrganization: { projects: { some: { id: input.projectId } } } },
                ]
            } : {}),
            (input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            (input.routineId !== undefined ? {
                OR: [
                    { routineInputs: { some: { routineId: input.routineId } } },
                    { routineOutputs: { some: { routineId: input.routineId } } },
                ]
            } : {}),
            (input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
            (!!input.type ? { type: { contains: input.type.trim(), mode: 'insensitive' } } : {}),
        ])
    },
})

export const standardValidator = (): Validator<StandardCreateInput, StandardUpdateInput, Standard, Prisma.standard_versionSelect, Prisma.standard_versionWhereInput> => ({
    permissionsSelect: { id: true, root: { select: { 
        permissions: true, 
        user: { select: { id: true } }, 
        organization: { select: { id: true, permissions: true } } } } 
    },
    profanityFields: ['name'],
})

export const standardQuerier = (prisma: PrismaType) => ({
    /**
     * Checks for existing standards with the same shape. Useful to avoid duplicates
     * @param data StandardCreateData to check
     * @param userId The ID of the user creating the standard
     * @param uniqueToCreator Whether to check if the standard is unique to the user/organization 
     * @param isInternal Used to determine if the standard should show up in search results
     * @returns data of matching standard, or null if no match
     */
    async findMatchingStandardVersion(
        data: StandardCreateInput,
        userId: string,
        uniqueToCreator: boolean,
        isInternal: boolean
    ): Promise<{ [x: string]: any } | null> {
        // Sort all JSON properties that are part of the comparison
        const props = sortify(data.props);
        const yup = data.yup ? sortify(data.yup) : null;
        // Find all standards that match the given standard
        const standards = await prisma.standard_version.findMany({
            where: {
                root: {
                    isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
                    isDeleted: false,
                    isPrivate: false,
                    createdByUserId: (uniqueToCreator && !data.createdByOrganizationId) ? userId : undefined,
                    createdByOrganizationId: (uniqueToCreator && data.createdByOrganizationId) ? data.createdByOrganizationId : undefined,
                },
                default: data.default ?? null,
                props: props,
                yup: yup,
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
     * createdByOrganizationId, and name
     * @param data StandardCreateData to check
     * @param userId The ID of the user creating the standard
     * @returns data of matching standard, or null if no match
     */
    async findMatchingStandardName(
        data: StandardCreateInput & { name: string },
        userId: string
    ): Promise<{ [x: string]: any } | null> {
        // Find all standards that match the given standard
        const standards = await prisma.standard.findMany({
            where: {
                name: data.name,
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
     * Standards must have a unique name per user/organization
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
                }
            });
            if (existing.length > 0) i++;
            else success = true;
        }
        return i === 0 ? name : `${name}${i}`;
    }
})

export const standardMutater = (prisma: PrismaType) => ({
    async shapeBase(userId: string, data: StandardCreateInput | StandardUpdateInput) {
        return {
            root: {
                isPrivate: data.isPrivate ?? undefined,
                tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Standard'),
            },
            isPrivate: data.isPrivate ?? undefined,
            resourceList: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, true),
        }
    },
    async shapeRelationshipCreate(userId: string, data: StandardCreateInput): Promise<Prisma.standard_versionUpsertArgs['create']> {
        let translations = TranslationModel.relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, true)
        if (translations?.jsonVariable) {
            translations.jsonVariable = sortify(translations.jsonVariable);
        }
        const base = await this.shapeBase(userId, data,);
        return {
            ...base,
            root: {
                create: {
                    ...base.root,
                    isInternal: data.isInternal ?? false,
                    name: await standardQuerier(prisma).generateName(userId, data),
                    permissions: JSON.stringify({}),
                },
            },
            default: data.default ?? undefined,
            versionLabel: data.versionLabel ?? '1.0.0',
            type: data.type,
            props: sortify(data.props),
            yup: data.yup ? sortify(data.yup) : undefined,
            translations,
        }
    },
    async shapeRelationshipUpdate(userId: string, data: StandardUpdateInput): Promise<Prisma.standard_versionUpsertArgs['update']> {
        let translations = TranslationModel.relationshipBuilder(userId, data, { create: standardTranslationCreate, update: standardTranslationUpdate }, false)
        if (translations?.jsonVariable) {
            translations.jsonVariable = sortify(translations.jsonVariable);
        }
        const base = await this.shapeBase(userId, data);
        return {
            ...base,
            root: {
                update: {
                    ...base.root,
                },
            },
            translations,
        }
    },
    async shapeCreate(userId: string, data: StandardCreateInput): Promise<Prisma.standard_versionUpsertArgs['create']> {
        const base: any = await this.shapeRelationshipCreate(userId, data);
        return {
            ...base,
            root: {
                create: {
                    ...base.root.create,
                    createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                    organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                    createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                    user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                },
            },
        } as any
    },
    async shapeUpdate(userId: string, data: StandardUpdateInput): Promise<Prisma.standard_versionUpsertArgs['update']> {
        const base: any = await this.shapeRelationshipUpdate(userId, data);
        return {
            ...base,
            root: {
                update: {
                    ...base.root.update,
                    organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                    user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
                },
            },
        } as AnyAaaaRecord
    },
    /**
     * Add, update, or remove a one-to-one standard relationship. 
     * Due to some unknown Prisma bug, it won't let us create/update a standard directly
     * in the main mutation query like most other relationship builders. Instead, we 
     * must do this separately, and return the standard's ID.
     */
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<string | null> {
        // Convert input to Prisma shape
        const fieldExcludes: string[] = [];
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName: 'standard', isAdd, fieldExcludes })
        // Shape
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            let create: any;
            // If standard is internal, check if the shape exists in the database
            if (formattedInput.create[0].isInternal) {
                const existingStandard = await standardQuerier(prisma).findMatchingStandardVersion(formattedInput.create[0], userId, false, true);
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
                create = await this.shapeCreate(userId, formattedInput.create[0]);
                const check1 = await standardQuerier(prisma).findMatchingStandardName(create.name, userId);
                if (check1) {
                    throw new CustomError(CODE.StandardDuplicateName, 'Standard with this name/version pair already exists.', { code: genErrorCode('0240') });
                }
                const check2 = await standardQuerier(prisma).findMatchingStandardVersion(create, userId, true, false)
                if (check2) {
                    return check2.id;
                }
            }
            // Shape create data
            if (!create) create = await this.shapeCreate(userId, formattedInput.create[0]);
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
            const update = await this.shapeUpdate(userId, formattedInput.update[0]);
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
                req: { users: [{ id: userId }] },
            })
            return deleteId;
        }
        return null;
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<StandardCreateInput, StandardUpdateInput>): Promise<CUDResult<Standard>> {
        const select = selectHelper(partialInfo);
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            let data: any;
            // Loop through each create input
            for (const input of createMany) {
                // If standard is internal, check if the shape exists in the database
                if (input.isInternal) {
                    const existingStandard = await standardQuerier(prisma).findMatchingStandardVersion(input, userId, false, true);
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
                    data = await this.shapeCreate(userId, input);
                    const check1 = await standardQuerier(prisma).findMatchingStandardName(data.name, userId);
                    if (check1) {
                        throw new CustomError(CODE.StandardDuplicateName, 'Standard with this name/version pair already exists.', { code: genErrorCode('0238') });
                    }
                    const check2 = await standardQuerier(prisma).findMatchingStandardVersion(data, userId, true, false)
                    if (check2) {
                        throw new CustomError(CODE.StandardDuplicateShape, 'Standard with this shape already exists.', { code: genErrorCode('0239') });
                    }
                }
                // If not called, create data
                if (!data) data = await this.shapeCreate(userId, input);
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
                    data: await this.shapeUpdate(userId, input.data),
                    ...select
                });
                // TODO handle version update
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
                        versions: {
                            select: {
                                _count: {
                                    select: {
                                        routineInputs: true,
                                        routineOutputs: true,
                                    }
                                }
                            }
                        }
                    }
                })
                // If standard not found, throw error
                if (!standard) {
                    throw new CustomError(CODE.NotFound, 'Standard not found', { code: genErrorCode('0241') });
                }
                // If standard is being used, anonymize
                if (standard.versions.some(v => v._count.routineInputs > 0 || v._count.routineOutputs > 0)) {
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

export const StandardModel = ({
    prismaObject: (prisma: PrismaType) => prisma.standard_version,
    format: standardFormatter(),
    mutate: standardMutater,
    query: standardQuerier,
    search: standardSearcher(),
    type: 'Standard' as GraphQLModelType,
    validate: standardValidator(),
})