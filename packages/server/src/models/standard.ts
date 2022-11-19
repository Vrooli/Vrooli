import { standardsCreate, standardsUpdate, standardTranslationCreate, standardTranslationUpdate } from "@shared/validation";
import { StandardSortBy } from "@shared/consts";
import { addCountFieldsHelper, addJoinTablesHelper, combineQueries, permissionsSelectHelper, relationshipBuilderHelper, removeCountFieldsHelper, removeJoinTablesHelper, visibilityBuilder } from "./builder";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { TranslationModel } from "./translation";
import { ViewModel } from "./view";
import { ResourceListModel } from "./resourceList";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, Mutater, Searcher, Validator } from "./types";
import { randomString } from "../auth/walletAuth";
import { Trigger } from "../events";
import { Standard, StandardPermission, StandardSearchInput, StandardCreateInput, StandardUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { sortify } from "../utils/objectTools";
import { Prisma } from "@prisma/client";
import { oneIsPublic } from "./utils";
import { organizationQuerier } from "./organization";
import { cudHelper } from "./actions";
import { getSingleTypePermissions } from "./validators";

const joinMapper = { tags: 'tag', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsStandard' | 'versions';
export const standardFormatter = (): FormatConverter<Standard, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Standard',
        comments: 'Comment',
        creator: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        reports: 'Report',
        resourceLists: 'ResourceList',
        routineInputs: 'Routine',
        routineOutputs: 'Routine',
        starredBy: 'User',
        tags: 'Tag',
    },
    rootFields: ['hasCompleteVersion', 'isDeleted', 'isInternal', 'isPrivate', 'name', 'votes', 'stars', 'views', 'permissions'],
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isUpvoted', 'isStarred', 'isViewed', 'permissionsStandard', 'versions'],
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query(prisma).getIsStarreds(userData?.id, ids, 'Standard')],
            ['isUpvoted', async () => await VoteModel.query(prisma).getIsUpvoteds(userData?.id, ids, 'Standard')],
            ['isViewed', async () => await ViewModel.query(prisma).getIsVieweds(userData?.id, ids, 'Standard')],
            ['permissionsStandard', async () => await getSingleTypePermissions('Standard', ids, prisma, userData)],
            ['versions', async () => {
                const groupData = await prisma.standard.findMany({
                    where: { id: { in: ids } },
                    select: { versions: { select: { id: true, versionLabel: true } } }
                });
                return groupData.map(g => g.versions);
            }],
        ],
    },
})

export const standardSearcher = (): Searcher<
    StandardSearchInput,
    StandardSortBy,
    Prisma.standard_versionOrderByWithRelationInput,
    Prisma.standard_versionWhereInput
> => ({
    defaultSort: StandardSortBy.VotesDesc,
    sortMap: {
        CommentsAsc: { comments: { _count: 'asc' } },
        CommentsDesc: { comments: { _count: 'desc' } },
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        StarsAsc: { root: { stars: 'asc' } },
        StarsDesc: { root: { stars: 'desc' } },
        VotesAsc: { root: { votes: 'asc' } },
        VotesDesc: { root: { votes: 'desc' } },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
            { name: { ...insensitive } },
            { tags: { some: { tag: { tag: { ...insensitive } } } } },
        ]
    }),
    customQueries(input, userId) {
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

export const standardValidator = (): Validator<
    StandardCreateInput,
    StandardUpdateInput,
    Standard,
    Prisma.standard_versionGetPayload<{ select: { [K in keyof Required<Prisma.standard_versionSelect>]: true } }>,
    StandardPermission,
    Prisma.standard_versionSelect,
    Prisma.standard_versionWhereInput
> => ({
    validateMap: {
        __typename: 'Standard',
        root: {
            select: {
                parent: 'Standard',
                organization: 'Organization',
                user: 'User',
            }
        },
        forks: 'Standard',
        // directoryListings: 'ProjectDirectory',
    },
    permissionsSelect: (userId) => ({
        id: true,
        isComplete: true,
        isPrivate: true,
        isDeleted: true,
        root: {
            select: {
                isInternal: true,
                isPrivate: true,
                isDeleted: true,
                permissions: true,
                ...permissionsSelectHelper([
                    ['organization', 'Organization'],
                    ['user', 'User'],
                ], userId)
            }
        }
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ([
        ['canComment', async () => !isDeleted && (isAdmin || isPublic)],
        ['canDelete', async () => isAdmin && !isDeleted],
        ['canEdit', async () => isAdmin && !isDeleted],
        ['canReport', async () => !isAdmin && !isDeleted && isPublic],
        ['canStar', async () => !isDeleted && (isAdmin || isPublic)],
        ['canView', async () => !isDeleted && (isAdmin || isPublic)],
        ['canVote', async () => !isDeleted && (isAdmin || isPublic)],
    ]),
    owner: (data) => ({
        Organization: (data.root as any).organization,
        User: (data.root as any).user,
    }),
    isDeleted: (data) => data.isDeleted || data.root.isDeleted,
    isPublic: (data) => data.isPrivate === false &&
        data.isDeleted === false &&
        data.root?.isDeleted === false &&
        data.root?.isInternal === false &&
        data.root?.isPrivate === false && oneIsPublic<Prisma.standardSelect>(data.root, [
            ['organization', 'Organization'],
            ['user', 'User'],
        ]),
    profanityFields: ['name'],
    ownerOrMemberWhere: (userId) => ({
        root: {
            OR: [
                organizationQuerier().hasRoleInOrganizationQuery(userId),
                { user: { id: userId } }
            ]
        }
    }),
    // TODO perform unique checks: Check if standard with same createdByUserId, createdByOrganizationId, name, and version already exists with the same creator
    // TODO when deleting, anonymize standards which are being used by inputs/outputs
    // const standard = await prisma.standard_version.findUnique({
    //     where: { id },
    //     select: {
    //                 _count: {
    //                     select: {
    //                         routineInputs: true,
    //                         routineOutputs: true,
    //                     }
    //                 }
    //     }
    // })
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

export const standardMutater = (prisma: PrismaType): Mutater<Standard> => ({
    async shapeBase(userId: string, data: StandardCreateInput | StandardUpdateInput) {
        return {
            root: {
                isPrivate: data.isPrivate ?? undefined,
                tags: await TagModel.mutate(prisma).tagRelationshipBuilder(userId, data, 'Standard'),
            },
            isPrivate: data.isPrivate ?? undefined,
            resourceList: await ResourceListModel.mutate(prisma).relationshipBuilder!(userId, data, true),
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
        }
    },
    /**
     * Add, update, or remove a one-to-one standard relationship. 
     * Due to some unknown Prisma bug, it won't let us create/update a standard directly
     * in the main mutation query like most other relationship builders. Instead, we 
     * must do this separately, and return the standard's ID.
     */
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Check if standard of same shape already exists with the same creator. 
        // If so, connect instead of creating a new standard
        const initialCreate = Array.isArray(data[`standardCreate`]) ?
            data[`standardCreate`].map((c: any) => c.id) :
            typeof data[`standardCreate`] === 'object' ? [data[`standardCreate`].id] :
                [];
        const initialConnect = Array.isArray(data[`standardConnect`]) ?
            data[`standardConnect`] :
            typeof data[`standardConnect`] === 'object' ? [data[`standardConnect`]] :
                [];
        const initialUpdate = Array.isArray(data[`standardUpdate`]) ?
            data[`standardUpdate`].map((c: any) => c.id) :
            typeof data[`standardUpdate`] === 'object' ? [data[`standardUpdate`].id] :
                [];
        const initialCombined = [...initialCreate, ...initialConnect, ...initialUpdate];
        // TODO this will have bugs
        if (initialCombined.length > 0) {
            const existingIds: string[] = [];
            // Find shapes of all initial standards
            for (const standard of initialCombined) {
                const initialShape = await this.shapeCreate(userId, standard);
                const exists = await standardQuerier(prisma).findMatchingStandardVersion(standard, userId, true, false)
                if (exists) existingIds.push(exists.id);
            }
            // All existing shapes are the new connects
            data[`standardConnect`] = existingIds;
            data[`standardCreate`] = initialCombined.filter((s: any) => !existingIds.includes(s.id));
        }
        return relationshipBuilderHelper({
            data,
            relationshipName: 'standard',
            isAdd,
            userId,
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
        });
    },
    async cud(params: CUDInput<StandardCreateInput, StandardUpdateInput>): Promise<CUDResult<Standard>> {
        return cudHelper({
            ...params,
            objectType: 'Standard',
            prisma,
            yup: { yupCreate: standardsCreate, yupUpdate: standardsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Standard', c.id as string, params.userData.id);
                }
            },
            onUpdated: (updatedData, updateInput) => {
                for (let i = 0; i < updatedData.length; i++) {
                    const u = updatedData[i];
                    const input = updateInput[i];
                    // Check if version changed
                    if (input.versionLabel && u.isComplete) {
                        Trigger(prisma).objectNewVersion('Standard', u.id as string, params.userData.id);
                    }
                }
            }
        })
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