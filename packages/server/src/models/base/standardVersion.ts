import { MaxObjects, SessionUser, StandardCreateInput, StandardVersionCreateInput, StandardVersionSortBy, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, getTranslation, standardVersionValidation } from "@local/shared";
import { ModelMap } from ".";
import { randomString } from "../../auth/codes";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { sortify } from "../../utils/objectTools";
import { PreShapeVersionResult, afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { StandardVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { StandardModelInfo, StandardModelLogic, StandardVersionModelInfo, StandardVersionModelLogic } from "./types";

type StandardVersionPre = PreShapeVersionResult;

//     // TODO perform unique checks: Check if standard with same createdByUserId, createdByTeamId, name, and version already exists with the same creator
//     //TODO when updating, not allowed to update existing, completed version
//     // TODO when deleting, anonymize standards which are being used by inputs/outputs
//     // const standard = await prismaInstance.standard_version.findUnique({
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

function querier() {
    return {
        /**
         * Checks for existing standards with the same shape. Useful to avoid duplicates
         * @param data StandardCreateData to check
         * @param userData The ID of the user creating the standard
         * @param uniqueToCreator Whether to check if the standard is unique to the user/team 
         * @param isInternal Used to determine if the standard should show up in search results
         * @returns data of matching standard, or null if no match
         */
        async findMatchingStandardVersion(
            data: StandardCreateInput,
            userData: SessionUser,
            uniqueToCreator: boolean,
            isInternal: boolean,
        ): Promise<{ [x: string]: any } | null> {
            return null;
            // // Sort all JSON properties that are part of the comparison
            // const props = sortify(data.props);
            // const yup = data.yup ? sortify(data.yup) : null;
            // // Find all standards that match the given standard
            // const standards = await prismaInstance.standard_version.findMany({
            //     where: {
            //         root: {
            //             isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
            //             isDeleted: false,
            //             isPrivate: false,
            //             createdByUserId: (uniqueToCreator && !data.createdByTeamId) ? userData.id : undefined,
            //             createdByTeamId: (uniqueToCreator && data.createdByTeamId) ? data.createdByTeamId : undefined,
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
         * @param userId The user's ID
         * @param languages The user's preferred languages
         * @param data The standard create data
         * @returns A valid name for the standard
         */
        async generateName(userId: string, languages: string[], data: StandardVersionCreateInput): Promise<string> {
            // First, check if name was already provided
            const translatedName = "";//getTranslation(data, 'name', languages).name ?? "";
            if (translatedName.length > 0) return translatedName;
            // Otherwise, generate name based on type and random string
            const name = `${data.variant} ${randomString(5)}`;
            return name;
        },
    };
}

const __typename = "StandardVersion" as const;
export const StandardVersionModel: StandardVersionModelLogic = ({
    __typename,
    dbTable: "standard_version",
    dbTranslationTable: "standard_version_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    name: trans.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans.description,
                }, languages?.[0]);
            },
        },
    }),
    format: StandardVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<StandardVersionPre> => {
                const { Create, Update, Delete, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    Update,
                });
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio"));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as StandardVersionPre;
                // If jsonVariables defined, sort them. 
                // This makes comparing standards a whole lot easier
                const translations = await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest });
                if (translations?.create?.length) {
                    translations.create = translations.create.map((t: StandardVersionTranslationCreateInput) => {
                        if (t.jsonVariable) t.jsonVariable = sortify(t.jsonVariable);
                        return t;
                    });
                }
                return {
                    id: data.id,
                    codeLanguage: data.codeLanguage,
                    default: noNull(data.default),
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    isFile: noNull(data.isFile),
                    props: sortify(data.props),
                    variant: data.variant,
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    yup: data.yup ? sortify(data.yup) : undefined,
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childStandardVersions", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "standardVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Standard", parentRelationshipName: "versions", data, ...rest }),
                    translations,
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as StandardVersionPre;
                // If jsonVariables defined, sort them
                const translations = await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest });
                if (translations?.update?.length) {
                    translations.update = translations.update.map((t: { where: { id: string }, data: StandardVersionTranslationUpdateInput }) => {
                        if (t.data.jsonVariable) t.data.jsonVariable = sortify(t.data.jsonVariable);
                        return t;
                    });
                }
                if (translations?.create?.length) {
                    translations.create = translations.create.map((t: StandardVersionTranslationCreateInput) => {
                        if (t.jsonVariable) t.jsonVariable = sortify(t.jsonVariable);
                        return t;
                    });
                }
                return {
                    codeLanguage: noNull(data.codeLanguage),
                    default: noNull(data.default),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    isFile: noNull(data.isFile),
                    props: data.props ? sortify(data.props) : undefined,
                    variant: noNull(data.variant),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    yup: data.yup ? sortify(data.yup) : undefined,
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childStandardVersions", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "standardVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Standard", parentRelationshipName: "versions", data, ...rest }),
                    translations,
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsVersion({ ...params, objectType: __typename });
            },
        },
        yup: standardVersionValidation,
    },
    query: querier(),
    search: {
        defaultSort: StandardVersionSortBy.DateCompletedDesc,
        sortBy: StandardVersionSortBy,
        searchFields: {
            codeLanguage: true,
            completedTimeFrame: true,
            createdByIdRoot: true,
            createdTimeFrame: true,
            isCompleteWithRoot: true,
            isInternalWithRoot: true,
            isLatest: true,
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            ownedByTeamIdRoot: true,
            ownedByUserIdRoot: true,
            reportId: true,
            rootId: true,
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            userId: true,
            variant: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
                { root: "nameWrapped" },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<StandardVersionModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<StandardVersionModelInfo["DbSelect"]>([["root", "Standard"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<StandardModelLogic>("Standard").validate().owner(data?.root as StandardModelInfo["DbModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Standard", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    root: useVisibility("Standard", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Objects you own
                        {
                            root: useVisibility("Standard", "Own", data),
                        },
                        // Public objects
                        {
                            isPrivate: false, // Can't be private
                            root: (useVisibility("StandardVersion", "Public", data) as { root: object }).root,
                        },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Private versions you own
                        {
                            isPrivate: true, // Version is private
                            root: useVisibility("Standard", "Own", data),
                        },
                        // Private roots you own
                        {
                            root: {
                                isPrivate: true, // Root is private
                                ...useVisibility("Standard", "Own", data),
                            },
                        },
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Public versions you own
                        {
                            isPrivate: false, // Version is public
                            root: useVisibility("Standard", "Own", data),
                        },
                        // Public roots you own
                        {
                            root: {
                                isPrivate: false, // Root is public
                                ...useVisibility("Standard", "Own", data),
                            },
                        },
                    ],
                };
            },
            public: function getPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false, // Version can't be private
                    root: {
                        isDeleted: false, // Root can't be deleted
                        isInternal: false, // Internal standards should never be in search results
                        isPrivate: false, // Root can't be private
                        OR: [
                            // Unowned
                            { ownedByTeam: null, ownedByUser: null },
                            // Owned by public teams
                            { ownedByTeam: useVisibility("Team", "Public", data) },
                            // Owned by public users
                            { ownedByUser: { isPrivate: false, isPrivateStandards: false } },
                        ],
                    },
                };
            },
        },
    }),
});
