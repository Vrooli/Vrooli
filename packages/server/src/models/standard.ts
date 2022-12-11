import { standardsCreate, standardsUpdate } from "@shared/validation";
import { StandardSortBy } from "@shared/consts";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Displayer, Formatter, Mutater, Searcher, Validator } from "./types";
import { randomString } from "../auth/wallet";
import { Trigger } from "../events";
import { Standard, StandardPermission, StandardSearchInput, StandardCreateInput, StandardUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { sortify } from "../utils/objectTools";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { relBuilderHelper } from "../actions";
import { getSingleTypePermissions } from "../validators";
import { padSelect, permissionsSelectHelper } from "../builders";
import { oneIsPublic, tagRelationshipBuilder, translationRelationshipBuilder } from "../utils";
import { StandardVersionModel } from "./standardVersion";
import { SelectWrap } from "../builders/types";

const __typename = 'Standard' as const;

const suppFields = ['isStarred', 'isUpvoted', 'isViewed', 'permissionsStandard'] as const;
const formatter = (): Formatter<Standard, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        comments: 'Comment',
        createdBy: 'User',
        owner: ['Organization', 'User'],
        reports: 'Report',
        resourceLists: 'ResourceList',
        routineInputs: 'Routine',
        routineOutputs: 'Routine',
        starredBy: 'User',
        tags: 'Tag',
    },
    joinMap: { tags: 'tag', starredBy: 'user' },
    countFields: ['commentsCount', 'reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename)],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename)],
            ['permissionsStandard', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<
    StandardSearchInput,
    StandardSortBy,
    Prisma.standardWhereInput
> => ({
    defaultSort: StandardSortBy.ScoreDesc,
    sortBy: StandardSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'issuesId',
        'labelsId',
        'minScore',
        'minStars',
        'minViews',
        'ownedByOrganizationId',
        'ownedByUserId',
        'parentId',
        'pullRequestsId',
        'questionsId',
        'standardTypeLatestVersion',
        'tags',
        'transfersId',
        'translationLanguagesLatestVersion',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
            { name: { ...insensitive } },
            { tags: { some: { tag: { tag: { ...insensitive } } } } },
        ]
    }),
    /**
     * isInternal routines should never appear in the query, since they are 
     * only meant for a single input/output
     */
    customQueryData: () => ({ isInternal: true }),
})

const validator = (): Validator<
    StandardCreateInput,
    StandardUpdateInput,
    Prisma.standardGetPayload<SelectWrap<Prisma.standardSelect>>,
    StandardPermission,
    Prisma.standardSelect,
    Prisma.standardWhereInput,
    true,
    true
> => ({
    validateMap: {
        __typename: 'Standard',
        parent: 'Standard',
        createdBy: 'User',
        ownedByOrganization: 'Organization',
        ownedByUser: 'User',
        versions: {
            select: {
                forks: 'Standard',
            }
        },
    },
    isTransferable: true,
    hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
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
    hasCompletedVersion: (data) => data.hasCompleteVersion === true,
    permissionsSelect: (...params) => ({
        id: true,
        isInternal: true,
        isPrivate: true,
        isDeleted: true,
        permissions: true,
        createdBy: padSelect({ id: true }),
        ...permissionsSelectHelper([
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], ...params),
        versions: {
            select: {
                isComplete: true,
                isPrivate: true,
                isDeleted: true,
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
        Organization: data.ownedByOrganization,
        User: data.ownedByUser,
    }),
    isDeleted: (data) => data.isDeleted,// || data.root.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false &&
        data.isDeleted === false &&
        data.isInternal === false &&
        //latest(data.versions)?.isPrivate === false &&
        //latest(data.versions)?.isDeleted === false &&
        oneIsPublic<Prisma.routineSelect>(data, [
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], languages),
    profanityFields: ['name'],
    visibility: {
        private: {
            isPrivate: true,
            // OR: [
            //     { isPrivate: true },
            //     { root: { isPrivate: true } },
            // ]
        },
        public: {
            isPrivate: false,
            // AND: [
            //     { isPrivate: false },
            //     { root: { isPrivate: false } },
            // ]
        },
        owner: (userId) => ({
            OR: [
                { ownedByUser: { id: userId } },
                { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            ]
            // root: {
            //     OR: [
            //         { ownedByUser: { id: userId } },
            //         { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            //     ]
            // }
        }),
    },
    // TODO perform unique checks: Check if standard with same createdByUserId, createdByOrganizationId, name, and version already exists with the same creator
    //TODO when updating, not allowed to update existing, completed version
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
        return null;
        // // Sort all JSON properties that are part of the comparison
        // const props = sortify(data.props, userData.languages);
        // const yup = data.yup ? sortify(data.yup, userData.languages) : null;
        // // Find all standards that match the given standard
        // const standards = await prisma.standard_version.findMany({
        //     where: {
        //         root: {
        //             isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
        //             isDeleted: false,
        //             isPrivate: false,
        //             createdByUserId: (uniqueToCreator && !data.createdByOrganizationId) ? userData.id : undefined,
        //             createdByOrganizationId: (uniqueToCreator && data.createdByOrganizationId) ? data.createdByOrganizationId : undefined,
        //         },
        //         default: data.default ?? null,
        //         props: props,
        //         yup: yup,
        //     }
        // });
        // // If any standards match (should only ever be 0 or 1, but you never know) return the first one
        // if (standards.length > 0) {
        //     return standards[0];
        // }
        // // If no standards match, then data is unique. Return null
        // return null;
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
        version: {
            isPrivate: data.isPrivate ?? undefined,
            resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
        },
    }
}

const mutater = (): Mutater<
    Standard,
    { graphql: StandardCreateInput, db: Prisma.standardUpsertArgs['create'] },
    { graphql: StandardUpdateInput, db: Prisma.standardUpsertArgs['update'] },
    { graphql: StandardCreateInput, db: Prisma.standardCreateWithoutTransfersInput },
    { graphql: StandardUpdateInput, db: Prisma.standardUpdateWithoutTransfersInput }
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
                ...base.root,
                isInternal: data.isInternal ?? undefined,
                name: data.name ?? await querier().generateName(prisma, userData.id, data),
                permissions: JSON.stringify({}), //TODO
                createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                versions: {
                    create: {
                        ...base.version,
                        default: data.default ?? undefined,
                        props: sortify(data.props, userData.languages),
                        yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
                        type: data.type,
                        translations,
                    }
                }
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
                ...base.root,
                organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
                // If versionId is provided, update that version. 
                // Otherwise, versionLabel is provided, so create new version with that label
                versions: {
                    ...(data.versionId ? {
                        update: {
                            where: { id: data.versionId },
                            data: {
                                ...base.version,
                                default: data.default ?? undefined,
                                props: data.props ? sortify(data.props, userData.languages) : undefined,
                                yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
                                type: data.type as string,
                                translations,
                            }
                        }
                    } : {
                        create: {
                            ...base.version,
                            versionLabel: data.versionLabel as string,
                            default: data.default as string,
                            props: sortify(data.props as string, userData.languages),
                            yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
                            type: data.type as string,
                            translations,
                        }
                    })
                },
            }
        },
        relCreate: (...args) => mutater().shape.create(...args),
        relUpdate: (...args) => mutater().shape.update(...args),
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).createStandard(userData.id, c.id as string);
            }
        },
        onUpdated: ({ prisma, updated, updateInput, userData }) => {
            // for (let i = 0; i < updated.length; i++) {
            //     const u = updated[i];
            //     const input = updateInput[i];
            //     // Check if version changed
            //     if (input.versionLabel && u.isComplete) {
            //         Trigger(prisma, userData.languages).objectNewVersion('Standard', u.id as string, userData.id);
            //     }
            // }
        },
    },
    yup: { create: standardsCreate, update: standardsUpdate },
    /**
     * Add, update, or remove a one-to-one standard relationship. 
     * Due to some unknown Prisma bug, it won't let us create/update a standard directly
     * in the main mutation query like most other relationship builders. Instead, we 
     * must do this separately, and return the standard's ID.
     */
    // async relationshipBuilder(
    //     userId: string,
    //     data: { [x: string]: any },
    //     isAdd: boolean = true,
    // ): Promise<{ [x: string]: any } | undefined> {
    //     // Check if standard of same shape already exists with the same creator. 
    //     // If so, connect instead of creating a new standard
    //     const initialCreate = Array.isArray(data[`standardCreate`]) ?
    //         data[`standardCreate`].map((c: any) => c.id) :
    //         typeof data[`standardCreate`] === 'object' ? [data[`standardCreate`].id] :
    //             [];
    //     const initialConnect = Array.isArray(data[`standardConnect`]) ?
    //         data[`standardConnect`] :
    //         typeof data[`standardConnect`] === 'object' ? [data[`standardConnect`]] :
    //             [];
    //     const initialUpdate = Array.isArray(data[`standardUpdate`]) ?
    //         data[`standardUpdate`].map((c: any) => c.id) :
    //         typeof data[`standardUpdate`] === 'object' ? [data[`standardUpdate`].id] :
    //             [];
    //     const initialCombined = [...initialCreate, ...initialConnect, ...initialUpdate];
    //     // TODO this will have bugs
    //     if (initialCombined.length > 0) {
    //         const existingIds: string[] = [];
    //         // Find shapes of all initial standards
    //         for (const standard of initialCombined) {
    //             const initialShape = await this.shapeCreate(userId, standard);
    //             const exists = await querier().findMatchingStandardVersion(prisma, standard, userId, true, false)
    //             if (exists) existingIds.push(exists.id);
    //         }
    //         // All existing shapes are the new connects
    //         data[`standardConnect`] = existingIds;
    //         data[`standardCreate`] = initialCombined.filter((s: any) => !existingIds.includes(s.id));
    //     }
    //     return relationshipBuilderHelper({
    //         data,
    //         relationshipName: 'standard',
    //         isAdd,
    //         userId,
    //         shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
    //     });
    // },
})

const displayer = (): Displayer<
    Prisma.standardSelect,
    Prisma.standardGetPayload<SelectWrap<Prisma.standardSelect>>
> => ({
    select: () => ({
        id: true,
        versions: {
            where: { isPrivate: false },
            orderBy: { versionIndex: 'desc' },
            take: 1,
            select: StandardVersionModel.display.select(),
        }
    }),
    label: (select, languages) => select.versions.length > 0 ?
        StandardVersionModel.display.label(select.versions[0] as any, languages) : '',
})

export const StandardModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.standard,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    query: querier(),
    search: searcher(),
    validate: validator(),
})