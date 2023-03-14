import { Displayer, Formatter, ModelLogic, Mutater, Searcher, Validator } from "./types";
import { randomString } from "../auth/wallet";
import { Trigger } from "../events";
import { Standard, StandardCreateInput, StandardUpdateInput, SessionUser, StandardVersionSortBy, StandardVersionSearchInput, StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput, VersionYou, PrependString, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { sortify } from "../utils/objectTools";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { noNull, selPad, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, onCommonVersion, oneIsPublic, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { StandardModel } from "./standard";
import { standardVersionValidation } from "@shared/validation";

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
        const translatedName = '';//bestLabel(data.translationsCreate ?? [], 'name', languages);
        if (translatedName.length > 0) return translatedName;
        // Otherwise, generate name based on type and random string
        const name = `${data.standardType} ${randomString(5)}`
        return name;
    }
})

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
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                // If jsonVariables defined, sort them. 
                // This makes comparing standards a whole lot easier
                const { translations } = await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest });
                if (translations?.create?.length) {
                    translations.create = translations.create.map(t => {
                        t.jsonVariables = sortify(t.jsonVariables, rest.userData.languages);
                        return t;
                    })
                }
                return {
                    id: data.id,
                    default: noNull(data.default),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    isFile: noNull(data.isFile),
                    props: sortify(data.props, rest.userData.languages),
                    standardType: data.standardType,
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    yup: data.yup ? sortify(data.yup, rest.userData.languages) : undefined,
                    ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childStandardVersions', data, ...rest })),
                    ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'standardVersion', data, ...rest })),
                    ...(await shapeHelper({ relation: 'root', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: true, objectType: 'Standard', parentRelationshipName: 'versions', data, ...rest })),
                    translations,
                }
            },
            update: async ({ data, ...rest }) => {
                // If jsonVariables defined, sort them
                const { translations } = await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest });
                if (translations?.update?.length) {
                    translations.update = translations.update.map(t => {
                        t.data = {
                            ...t.data,
                            jsonVariables: sortify(t.data.jsonVariables, rest.userData.languages),
                        }
                        return t;
                    })
                }
                if (translations?.create?.length) {
                    translations.create = translations.create.map(t => {
                        t.jsonVariables = sortify(t.jsonVariables, rest.userData.languages);
                        return t;
                    })
                }
                return {
                    default: noNull(data.default),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    isFile: noNull(data.isFile),
                    props: data.props ? sortify(data.props, rest.userData.languages) : undefined,
                    standardType: noNull(data.standardType),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    yup: data.yup ? sortify(data.yup, rest.userData.languages) : undefined,
                    ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childStandardVersions', data, ...rest })),
                    ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'standardVersion', data, ...rest })),
                    ...(await shapeHelper({ relation: 'root', relTypes: ['Update'], isOneToOne: true, isRequired: true, objectType: 'Standard', parentRelationshipName: 'versions', data, ...rest })),
                    translations,
                }
            },
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonVersion({ ...params, objectType: __typename });
            },
        },
        yup: standardVersionValidation,
    },
    query: querier(),
    search: {
        defaultSort: StandardVersionSortBy.DateCompletedDesc,
        sortBy: StandardVersionSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdByIdRoot: true,
            createdTimeFrame: true,
            isCompleteWithRoot: true,
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            ownedByOrganizationIdRoot: true,
            ownedByUserIdRoot: true,
            reportId: true,
            rootId: true,
            standardType: true,
            tagsRoot: true,
            translationLanguages: true,
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
        maxObjects: MaxObjects[__typename],
        owner: (data) => StandardModel.validate!.owner(data.root as any),
        permissionsSelect: () => ({
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
                isDeleted: false,
                root: { isDeleted: false },
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ]
            },
            public: {
                isDeleted: false,
                root: { isDeleted: false },
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