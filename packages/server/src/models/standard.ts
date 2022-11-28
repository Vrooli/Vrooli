import { standardsCreate, standardsUpdate } from "@shared/validation";
import { StandardSortBy } from "@shared/consts";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Displayer, Formatter, GraphQLModelType, Mutater, Searcher, Validator } from "./types";
import { randomString } from "../auth/wallet";
import { Trigger } from "../events";
import { Standard, StandardPermission, StandardSearchInput, StandardCreateInput, StandardUpdateInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { sortify } from "../utils/objectTools";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { relBuilderHelper } from "../actions";
import { getSingleTypePermissions } from "../validators";
import { combineQueries, permissionsSelectHelper, visibilityBuilder } from "../builders";
import { oneIsPublic, tagRelationshipBuilder, translationRelationshipBuilder } from "../utils";

type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsStandard' | 'versions';
const formatter = (): Formatter<Standard, SupplementalFields> => ({
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
    joinMap: { tags: 'tag', starredBy: 'user' },
    countMap: { commentsCount: 'comments', reportsCount: 'reports' },
    supplemental: {
        graphqlFields: ['isUpvoted', 'isStarred', 'isViewed', 'permissionsStandard', 'versions'],
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'Standard')],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, 'Standard')],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, 'Standard')],
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

const searcher = (): Searcher<
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
    customQueries(input, userData) {
        return combineQueries([
            /**
             * isInternal routines should never appear in the query, since they are 
             * only meant for a single input/output
             */
            { isInternal: false },
            visibilityBuilder({ objectType: 'Standard', userData, visibility: input.visibility }),
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

const validator = (): Validator<
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
                createdBy: 'User',
                ownedByOrganization: 'Organization',
                ownedByUser: 'User',
            }
        },
        forks: 'Standard',
        // directoryListings: 'ProjectDirectory',
    },
    isTransferable: true,
    maxObjects: {
        User: {
            private: {
                noPremium: 5,
                premium: 100,
            },
            public: {
                noPremium: 100,
                premium: 1000,
            },
        },
        Organization: {
            private: {
                noPremium: 5,
                premium: 100,
            },
            public: {
                noPremium: 100,
                premium: 1000,
            },
        },
    },
    permissionsSelect: (...params) => ({
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
                    ['ownedByOrganization', 'Organization'],
                    ['ownedByUser', 'User'],
                ], ...params)
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
        Organization: (data.root as any).ownedByOrganization,
        User: (data.root as any).ownedByUser,
    }),
    isDeleted: (data) => data.isDeleted || data.root.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false &&
        data.isDeleted === false &&
        data.root?.isDeleted === false &&
        data.root?.isInternal === false &&
        data.root?.isPrivate === false && oneIsPublic<Prisma.standardSelect>(data.root, [
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], languages),
    profanityFields: ['name'],
    visibility: {
        private: {
            OR: [
                { isPrivate: true },
                { root: { isPrivate: true } },
            ]
        },
        public: {
            AND: [
                { isPrivate: false },
                { root: { isPrivate: false } },
            ]
        },
        owner: (userId) => ({
            root: {
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }
        }),
    },
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

const querier = () => ({
    /**
     * Checks for existing standards with the same shape. Useful to avoid duplicates
     * @param prisma Prisma client
     * @param data StandardCreateData to check
     * @param userData The ID of the user creating the standard
     * @param uniqueToCreator Whether to check if the standard is unique to the user/organization 
     * @param isInternal Used to determine if the standard should show up in search results
     * @returns data of matching standard, or null if no match
     */
    async findMatchingStandardVersion(
        prisma: PrismaType,
        data: StandardCreateInput,
        userData: SessionUser,
        uniqueToCreator: boolean,
        isInternal: boolean
    ): Promise<{ [x: string]: any } | null> {
        // Sort all JSON properties that are part of the comparison
        const props = sortify(data.props, userData.languages);
        const yup = data.yup ? sortify(data.yup, userData.languages) : null;
        // Find all standards that match the given standard
        const standards = await prisma.standard_version.findMany({
            where: {
                root: {
                    isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
                    isDeleted: false,
                    isPrivate: false,
                    createdByUserId: (uniqueToCreator && !data.createdByOrganizationId) ? userData.id : undefined,
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
     * @param prisma Prisma client
     * @param data StandardCreateData to check
     * @param userId The ID of the user creating the standard
     * @returns data of matching standard, or null if no match
     */
    async findMatchingStandardName(
        prisma: PrismaType,
        data: StandardCreateInput & { name: string },
        userId: string
    ): Promise<{ [x: string]: any } | null> {
        // Find all standards that match the given standard
        const standards = await prisma.standard.findMany({
            where: {
                name: data.name,
                ownedByUserId: !data.createdByOrganizationId ? userId : undefined,
                ownedByOrganizationId: data.createdByOrganizationId ? data.createdByOrganizationId : undefined,
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
     * @param prisma Prisma client
     * @param userId The user's ID
     * @param data The standard create data
     * @returns A valid name for the standard
     */
    async generateName(prisma: PrismaType, userId: string, data: StandardCreateInput): Promise<string> {
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

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: StandardCreateInput | StandardUpdateInput, isAdd: boolean) => {
    return {
        root: {
            isPrivate: data.isPrivate ?? undefined,
            tags: await tagRelationshipBuilder(prisma, userData, data, 'Standard', isAdd),
        },
        isPrivate: data.isPrivate ?? undefined,
        resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
    }
}

const mutater = (): Mutater<
    Standard,
    { graphql: StandardCreateInput, db: Prisma.standard_versionUpsertArgs['create'] },
    { graphql: StandardUpdateInput, db: Prisma.standard_versionUpsertArgs['update'] },
    { graphql: StandardCreateInput, db: Prisma.standard_versionCreateWithoutRootInput },
    { graphql: StandardUpdateInput, db: Prisma.standard_versionUpdateWithoutRootInput }
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            const base = await shapeBase(prisma, userData, data, true);
            // If jsonVariables defined, sort them
            let translations = await translationRelationshipBuilder(prisma, userData, data, true)
            if (translations?.create?.length) {
                translations.create = translations.create.map(t => {
                    t.jsonVariables = sortify(t.jsonVariables, userData.languages);
                    return t;
                })
            }
            return {
                ...base,
                root: {
                    create: {
                        ...base.root,
                        isInternal: data.isInternal ?? undefined,
                        name: data.name ?? await querier().generateName(prisma, userData.id, data),
                        permissions: JSON.stringify({}), //TODO
                        createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                        organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                        createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                        user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                    },
                },
                default: data.default ?? undefined,
                props: sortify(data.props, userData.languages),
                yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
                type: data.type,
                translations,
            }
        },
        update: async ({ data, prisma, userData }) => {
            const base = await shapeBase(prisma, userData, data, false);
            // If jsonVariables defined, sort them
            let translations = await translationRelationshipBuilder(prisma, userData, data, false)
            if (translations?.update?.length) {
                translations.update = translations.update.map(t => {
                    t.data = {
                        ...t.data,
                        jsonVariables: sortify(t.data.jsonVariables, userData.languages),
                    }
                    return t;
                })
            }
            if (translations?.create?.length) {
                translations.create = translations.create.map(t => {
                    t.jsonVariables = sortify(t.jsonVariables, userData.languages);
                    return t;
                })
            }
            return {
                ...base,
                root: {
                    update: {
                        ...base.root,
                        organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                        user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
                    },
                },
                translations,
            }
        },
        relCreate: mutater().shape.create,
        relUpdate: mutater().shape.update,
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).createStandard(userData.id, c.id as string);
            }
        },
        onUpdated: ({ prisma, updated, updateInput, userData }) => {
            for (let i = 0; i < updated.length; i++) {
                const u = updated[i];
                const input = updateInput[i];
                // Check if version changed
                if (input.versionLabel && u.isComplete) {
                    Trigger(prisma, userData.languages).objectNewVersion('Standard', u.id as string, userData.id);
                }
            }
        },
    },
    yup: { create: standardsCreate, update: standardsUpdate },
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
                const exists = await querier().findMatchingStandardVersion(prisma, standard, userId, true, false)
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
})

const displayer = (): Displayer<
    Prisma.standard_versionSelect,
    Prisma.standard_versionGetPayload<{ select: { [K in keyof Required<Prisma.standard_versionSelect>]: true } }>
> => ({
    select: { id: true, root: { select: { name: true }}},
    label: (select) => select.root.name ?? '',
})

export const StandardModel = ({
    delegate: (prisma: PrismaType) => prisma.standard_version,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    query: querier(),
    search: searcher(),
    type: 'Standard' as GraphQLModelType,
    validate: validator(),
})