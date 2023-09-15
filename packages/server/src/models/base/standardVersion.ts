import { MaxObjects, StandardCreateInput, StandardVersionCreateInput, StandardVersionSortBy, standardVersionValidation } from "@local/shared";
import { randomString } from "../../auth/wallet";
import { noNull, shapeHelper } from "../../builders";
import { PrismaType, SessionUserToken } from "../../types";
import { bestTranslation, defaultPermissions, getEmbeddableString, postShapeVersion, translationShapeHelper } from "../../utils";
import { sortify } from "../../utils/objectTools";
import { preShapeVersion } from "../../utils/preShapeVersion";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { StandardVersionFormat } from "../format/standardVersion";
import { ModelLogic } from "../types";
import { StandardModel } from "./standard";
import { StandardModelLogic, StandardVersionModelLogic } from "./types";

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
        userData: SessionUserToken,
        uniqueToCreator: boolean,
        isInternal: boolean,
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
        const translatedName = "";//bestTranslation(data.translationsCreate ?? [], 'name', languages)?.name ?? "";
        if (translatedName.length > 0) return translatedName;
        // Otherwise, generate name based on type and random string
        const name = `${data.standardType} ${randomString(5)}`;
        return name;
    },
});

const __typename = "StandardVersion" as const;
const suppFields = ["you"] as const;
export const StandardVersionModel: ModelLogic<StandardVersionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.standard_version,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    name: trans?.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans?.description,
                }, languages[0]);
            },
        },
    },
    format: StandardVersionFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const { Create, Update, Delete, prisma, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    prisma,
                    Update,
                    userData,
                });
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                // If jsonVariables defined, sort them. 
                // This makes comparing standards a whole lot easier
                const { translations } = await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest });
                if (translations?.create?.length) {
                    translations.create = translations.create.map(t => {
                        t.jsonVariables = sortify(t.jsonVariables, rest.userData.languages);
                        return t;
                    });
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
                    ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childStandardVersions", data, ...rest })),
                    ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "standardVersion", data, ...rest })),
                    ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Standard", parentRelationshipName: "versions", data, ...rest })),
                    translations,
                };
            },
            update: async ({ data, ...rest }) => {
                // If jsonVariables defined, sort them
                const { translations } = await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest });
                if (translations?.update?.length) {
                    translations.update = translations.update.map(t => {
                        t.data = {
                            ...t.data,
                            jsonVariables: sortify(t.data.jsonVariables, rest.userData.languages),
                        };
                        return t;
                    });
                }
                if (translations?.create?.length) {
                    translations.create = translations.create.map(t => {
                        t.jsonVariables = sortify(t.jsonVariables, rest.userData.languages);
                        return t;
                    });
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
                    ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childStandardVersions", data, ...rest })),
                    ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "standardVersion", data, ...rest })),
                    ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: true, objectType: "Standard", parentRelationshipName: "versions", data, ...rest })),
                    translations,
                };
            },
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
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
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
                { root: "nameWrapped" },
            ],
        }),
        /**
         * isInternal routines should never appear in the query, since they are 
         * only meant for a single input/output
         */
        customQueryData: () => ({ root: { isInternal: true } }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            StandardModel.validate.isPublic(data.root as StandardModelLogic["PrismaModel"], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => StandardModel.validate.owner(data.root as StandardModelLogic["PrismaModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Standard", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: {
                isDeleted: false,
                root: { isDeleted: false },
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ],
            },
            public: {
                isDeleted: false,
                root: { isDeleted: false },
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ],
            },
            owner: (userId) => ({
                root: StandardModel.validate.visibility.owner(userId),
            }),
        },
    },
});
