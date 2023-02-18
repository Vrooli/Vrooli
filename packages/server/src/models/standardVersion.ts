import { Displayer, Formatter, ModelLogic, Mutater, Searcher, Validator } from "./types";
import { randomString } from "../auth/wallet";
import { Trigger } from "../events";
import { Standard, StandardCreateInput, StandardUpdateInput, SessionUser, StandardVersionSortBy, StandardVersionSearchInput, StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput, VersionYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { sortify } from "../utils/objectTools";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { selPad } from "../builders";
import { bestLabel, defaultPermissions, oneIsPublic } from "../utils";
import { SelectWrap } from "../builders/types";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { StandardModel } from "./standard";

// const validator = (): Validator<Model> => ({
//     validateMap: {
//         type: 'Standard',
//         parent: 'Standard',
//         createdBy: 'User',
//         ownedByOrganization: 'Organization',
//         ownedByUser: 'User',
//         versions: {
//             select: {
//                 forks: 'Standard',
//             }
//         },
//     },
//     isTransferable: true,
//     hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
//     maxObjects: {
//         User: {
//             private: {
//                 noPremium: 5,
//                 premium: 100,
//             },
//             public: {
//                 noPremium: 100,
//                 premium: 1000,
//             },
//         },
//         Organization: {
//             private: {
//                 noPremium: 5,
//                 premium: 100,
//             },
//             public: {
//                 noPremium: 100,
//                 premium: 1000,
//             },
//         },
//     },
//     hasCompleteVersion: (data) => data.hasCompleteVersion === true,
//     permissionsSelect: (...params) => ({
//         id: true,
//         isInternal: true,
//         isPrivate: true,
//         isDeleted: true,
//         permissions: true,
//         createdBy: selPad({ id: true }),
//         ...permissionsSelectHelper([
//             ['ownedByOrganization', 'Organization'],
//             ['ownedByUser', 'User'],
//         ], ...params),
//         versions: {
//             select: {
//                 isComplete: true,
//                 isPrivate: true,
//                 isDeleted: true,
//             }
//         }
//     }),
//     permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
//         canComment: () => !isDeleted && (isAdmin || isPublic),
//         canDelete: () => isAdmin && !isDeleted,
//         canUpdate: () => isAdmin && !isDeleted,
//         canReport: () => !isAdmin && !isDeleted && isPublic,
//         canBookmark: () => !isDeleted && (isAdmin || isPublic),
//         canView: () => !isDeleted && (isAdmin || isPublic),
//         canVote: () => !isDeleted && (isAdmin || isPublic),
//     }),
//     owner: (data) => ({
//         Organization: data.ownedByOrganization,
//         User: data.ownedByUser,
//     }),
//     isDeleted: (data) => data.isDeleted,// || data.root.isDeleted,
//     isPublic: (data, languages) => data.isPrivate === false &&
//         data.isDeleted === false &&
//         data.isInternal === false &&
//         //latest(data.versions)?.isPrivate === false &&
//         //latest(data.versions)?.isDeleted === false &&
//         oneIsPublic<Prisma.routineSelect>(data, [
//             ['ownedByOrganization', 'Organization'],
//             ['ownedByUser', 'User'],
//         ], languages),
//     profanityFields: ['name'],
//     visibility: {
//         private: {
//             isPrivate: true,
//             // OR: [
//             //     { isPrivate: true },
//             //     { root: { isPrivate: true } },
//             // ]
//         },
//         public: {
//             isPrivate: false,
//             // AND: [
//             //     { isPrivate: false },
//             //     { root: { isPrivate: false } },
//             // ]
//         },
//         owner: (userId) => ({
//             OR: [
//                 { ownedByUser: { id: userId } },
//                 { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
//             ]
//             // root: {
//             //     OR: [
//             //         { ownedByUser: { id: userId } },
//             //         { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
//             //     ]
//             // }
//         }),
//     },
//     // TODO perform unique checks: Check if standard with same createdByUserId, createdByOrganizationId, name, and version already exists with the same creator
//     //TODO when updating, not allowed to update existing, completed version
//     // TODO when deleting, anonymize standards which are being used by inputs/outputs
//     // const standard = await prisma.standard_version.findUnique({
//     //     where: { id },
//     //     select: {
//     //                 _count: {
//     //                     select: {
//     //                         routineInputs: true,
//     //                         routineOutputs: true,
//     //                     }
//     //                 }
//     //     }
//     // })
// })

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
     * Generates a name for a standard.
     * @param prisma Prisma client
     * @param userId The user's ID
     * @param languages The user's preferred languages
     * @param data The standard create data
     * @returns A valid name for the standard
     */
    async generateName(prisma: PrismaType, userId: string, languages: string[], data: StandardVersionCreateInput): Promise<string> {
        // First, check if name was already provided
        const translatedName = '' ;//bestLabel(data.translationsCreate ?? [], 'name', languages);
        if (translatedName.length > 0) return translatedName;
        // Otherwise, generate name based on type and random string
        const name = `${data.standardType} ${randomString(5)}`
        return name;
    }
})

// const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: StandardCreateInput | StandardUpdateInput, isAdd: boolean) => {
//     return {
//         root: {
//             isPrivate: data.isPrivate ?? undefined,
//             tags: await tagRelationshipBuilder(prisma, userData, data, 'Standard', isAdd),
//         },
//         version: {
//             isPrivate: data.isPrivate ?? undefined,
//             resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
//         },
//     }
// }

// const mutater = (): Mutater<Model> => ({
//     shape: {
//         create: async ({ data, prisma, userData }) => {
//             const base = await shapeBase(prisma, userData, data, true);
//             // If jsonVariables defined, sort them
//             let translations = await translationRelationshipBuilder(prisma, userData, data, true)
//             if (translations?.create?.length) {
//                 translations.create = translations.create.map(t => {
//                     t.jsonVariables = sortify(t.jsonVariables, userData.languages);
//                     return t;
//                 })
//             }
//             return {
//                 ...base.root,
//                 isInternal: data.isInternal ?? undefined,
//                 name: data.name ?? await querier().generateName(prisma, userData.id, data),
//                 permissions: JSON.stringify({}), //TODO
//                 createdByOrganization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
//                 organization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
//                 createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
//                 user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
//                 versions: {
//                     create: {
//                         ...base.version,
//                         default: data.default ?? undefined,
//                         props: sortify(data.props, userData.languages),
//                         yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
//                         standardType: data.standardType,
//                         translations,
//                     }
//                 }
//             }
//         },
//         update: async ({ data, prisma, userData }) => {
//             const base = await shapeBase(prisma, userData, data, false);
//             // If jsonVariables defined, sort them
//             let translations = await translationRelationshipBuilder(prisma, userData, data, false)
//             if (translations?.update?.length) {
//                 translations.update = translations.update.map(t => {
//                     t.data = {
//                         ...t.data,
//                         jsonVariables: sortify(t.data.jsonVariables, userData.languages),
//                     }
//                     return t;
//                 })
//             }
//             if (translations?.create?.length) {
//                 translations.create = translations.create.map(t => {
//                     t.jsonVariables = sortify(t.jsonVariables, userData.languages);
//                     return t;
//                 })
//             }
//             return {
//                 ...base.root,
//                 organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
//                 user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
//                 // If versionId is provided, update that version. 
//                 // Otherwise, versionLabel is provided, so create new version with that label
//                 versions: {
//                     ...(data.versionId ? {
//                         update: {
//                             where: { id: data.versionId },
//                             data: {
//                                 ...base.version,
//                                 default: data.default ?? undefined,
//                                 props: data.props ? sortify(data.props, userData.languages) : undefined,
//                                 yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
//                                 standardType: data.standardType as string,
//                                 translations,
//                             }
//                         }
//                     } : {
//                         create: {
//                             ...base.version,
//                             versionLabel: data.versionLabel as string,
//                             default: data.default as string,
//                             props: sortify(data.props as string, userData.languages),
//                             yup: data.yup ? sortify(data.yup, userData.languages) : undefined,
//                             standardType: data.standardType as string,
//                             translations,
//                         }
//                     })
//                 },
//             }
//         },
//     },
//     trigger: {
//         onCreated: ({ created, prisma, userData }) => {
//             for (const c of created) {
//                 Trigger(prisma, userData.languages).createStandard(userData.id, c.id as string);
//             }
//         },
//         onUpdated: ({ prisma, updated, updateInput, userData }) => {
//             // for (let i = 0; i < updated.length; i++) {
//             //     const u = updated[i];
//             //     const input = updateInput[i];
//             //     // Check if version changed
//             //     if (input.versionLabel && u.isComplete) {
//             //         Trigger(prisma, userData.languages).objectNewVersion('Standard', u.id as string, userData.id);
//             //     }
//             // }
//         },
//     },
//     yup: standardVersionValidation,
//     /**
//      * Add, update, or remove a one-to-one standard relationship. 
//      * Due to some unknown Prisma bug, it won't let us create/update a standard directly
//      * in the main mutation query like most other relationship builders. Instead, we 
//      * must do this separately, and return the standard's ID.
//      */
//     // async relationshipBuilder(
//     //     userId: string,
//     //     data: { [x: string]: any },
//     //     isAdd: boolean = true,
//     // ): Promise<{ [x: string]: any } | undefined> {
//     //     // Check if standard of same shape already exists with the same creator. 
//     //     // If so, connect instead of creating a new standard
//     //     const initialCreate = Array.isArray(data[`standardCreate`]) ?
//     //         data[`standardCreate`].map((c: any) => c.id) :
//     //         typeof data[`standardCreate`] === 'object' ? [data[`standardCreate`].id] :
//     //             [];
//     //     const initialConnect = Array.isArray(data[`standardConnect`]) ?
//     //         data[`standardConnect`] :
//     //         typeof data[`standardConnect`] === 'object' ? [data[`standardConnect`]] :
//     //             [];
//     //     const initialUpdate = Array.isArray(data[`standardUpdate`]) ?
//     //         data[`standardUpdate`].map((c: any) => c.id) :
//     //         typeof data[`standardUpdate`] === 'object' ? [data[`standardUpdate`].id] :
//     //             [];
//     //     const initialCombined = [...initialCreate, ...initialConnect, ...initialUpdate];
//     //     // TODO this will have bugs
//     //     if (initialCombined.length > 0) {
//     //         const existingIds: string[] = [];
//     //         // Find shapes of all initial standards
//     //         for (const standard of initialCombined) {
//     //             const initialShape = await this.shapeCreate(userId, standard);
//     //             const exists = await querier().findMatchingStandardVersion(prisma, standard, userId, true, false)
//     //             if (exists) existingIds.push(exists.id);
//     //         }
//     //         // All existing shapes are the new connects
//     //         data[`standardConnect`] = existingIds;
//     //         data[`standardCreate`] = initialCombined.filter((s: any) => !existingIds.includes(s.id));
//     //     }
//     //     return relationshipBuilderHelper({
//     //         data,
//     //         relationshipName: 'standard',
//     //         isAdd,
//     //         userId,
//     //         shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
//     //     });
//     // },
// })

const __typename = 'StandardVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canUpdate' | 'canReport' | 'canUse' | 'canRead'>;
const suppFields = ['you'] as const;
export const StandardVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: StandardVersionCreateInput,
    GqlUpdate: StandardVersionUpdateInput,
    GqlModel: StandardVersion,
    GqlSearch: StandardVersionSearchInput,
    GqlSort: StandardVersionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.standard_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.standard_versionUpsertArgs['update'],
    PrismaModel: Prisma.standard_versionGetPayload<SelectWrap<Prisma.standard_versionSelect>>,
    PrismaSelect: Prisma.standard_versionSelect,
    PrismaWhere: Prisma.standard_versionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.standard_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'StandardVersion',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'Standard',
        },
        prismaRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'StandardVersion',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'Standard',
        },
        countFields: {
            commentsCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
            },
        },
    },
    mutate: {} as any, //mutater(),
    query: querier(),
    search: {
        defaultSort: StandardVersionSortBy.DateCompletedDesc,
        sortBy: StandardVersionSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdTimeFrame: true,
            isCompleteWithRoot: true,
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            reportId: true,
            rootId: true,
            standardType: true,
            tagsRoot: true,
            updatedTimeFrame: true,
            userId: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                { root: 'tagsWrapped' },
                { root: 'labelsWrapped' },
                { root: 'nameWrapped' },
            ]
        }),
        /**
         * isInternal routines should never appear in the query, since they are 
         * only meant for a single input/output
         */
        customQueryData: () => ({ root: { isInternal: true } }),
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            StandardModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: 1000000,
        owner: (data) => StandardModel.validate!.owner(data.root as any),
        permissionsSelect: (...params) => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ['Standard', ['versions']],
        }),
        permissionResolvers: defaultPermissions,
        validations: {
            async common({ createMany, deleteMany, languages, prisma, updateMany }) {
                await versionsCheck({ 
                    createMany,
                    deleteMany,
                    languages,
                    objectType: 'Standard', 
                    prisma, 
                    updateMany: updateMany as any,
                });
            },
            async create({ createMany, languages }) {
                createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksBio', languages))
            },
            async update({ languages, updateMany }) {
                updateMany.forEach(({ data }) => lineBreaksCheck(data, ['description'], 'LineBreaksBio', languages));
            },
        },
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
                root: StandardModel.validate!.visibility.owner(userId),
            }),
        },
    },
})