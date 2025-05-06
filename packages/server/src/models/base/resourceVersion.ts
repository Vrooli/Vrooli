import { MaxObjects, ResourceVersionSortBy, generatePublicId, getTranslation, resourceVersionValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { EmbeddingService } from "../../services/embedding.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { calculateWeightData, type SubroutineWeightData } from "../../utils/routineComplexity.js";
import { afterMutationsVersion } from "../../utils/shapes/afterMutationsVersion.js";
import { preShapeVersion, type PreShapeVersionResult } from "../../utils/shapes/preShapeVersion.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { lineBreaksCheck } from "../../validators/lineBreaksCheck.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { versionsCheck } from "../../validators/versionsCheck.js";
import { ResourceVersionFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { ResourceModelInfo, ResourceModelLogic, ResourceVersionModelInfo, ResourceVersionModelLogic } from "./types.js";

type ResourceVersionPre = PreShapeVersionResult & {
    /** Map of routine version ID to graph complexity metrics */
    weightMap: Record<string, SubroutineWeightData & { id: string; }>;
};

const __typename = "ResourceVersion" as const;
export const ResourceVersionModel: ResourceVersionModelLogic = ({
    __typename,
    dbTable: "resource_version",
    dbTranslationTable: "resource_version_translation",
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
                return EmbeddingService.getEmbeddableString({
                    name: trans.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans.description,
                }, languages?.[0]);
            },
        },
    }),
    format: ResourceVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<ResourceVersionPre> => {
                const { Create, Update, Delete, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    Update,
                    userData,
                });
                const combinedInputs = [...Create, ...Update].map(d => d.input);
                combinedInputs.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio"));
                // Calculate simplicity and complexity of all versions. Since these calculations 
                // can depend on other versions, we need to do them all at once. 
                // We exclude deleting versions to ensure that they don't affect the calculations. 
                // If a deleting version appears in the calculations, an error will be thrown.
                const { dataWeights } = await calculateWeightData(
                    combinedInputs,
                    Delete.map(d => d.input),
                );
                // Convert dataWeights to a map for easy lookup
                const weightMap: ResourceVersionPre["weightMap"] = dataWeights.reduce((acc, curr) => {
                    acc[curr.id] = curr;
                    return acc;
                }, {});
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps, weightMap };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ResourceVersionPre;
                return {
                    id: BigInt(data.id),
                    publicId: rest.isSeeding ? (data.publicId ?? generatePublicId()) : generatePublicId(),
                    simplicity: preData.weightMap[data.id]?.simplicity ?? 0,
                    complexity: preData.weightMap[data.id]?.complexity ?? 0,
                    config: noNull(data.config),
                    isAutomatable: noNull(data.isAutomatable),
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    resourceSubType: data.resourceSubType,
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Resource", parentRelationshipName: "versions", data, ...rest }),
                    relatedVersions: await shapeHelper({
                        relation: "relatedVersions",
                        relTypes: ["Create"],
                        isOneToOne: false,
                        objectType: "ResourceVersion",
                        parentRelationshipName: "",
                        data,
                        //TODO
                        joinData: {
                            fieldName: "id",
                            uniqueFieldName: "routine_version_subroutine_parentRoutineId_subroutineId_unique",
                            childIdFieldName: "subroutineId",
                            parentIdFieldName: "parentRoutineId",
                            parentId: data.id ?? null,
                        },
                        ...rest,
                    }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ResourceVersionPre;
                return {
                    simplicity: preData.weightMap[data.id]?.simplicity ?? 0,
                    complexity: preData.weightMap[data.id]?.complexity ?? 0,
                    config: noNull(data.config),
                    isAutomatable: noNull(data.isAutomatable),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Resource", parentRelationshipName: "versions", data, ...rest }),
                    //TODO
                    relatedVersions: await shapeHelper({
                        relation: "relatedVersions",
                        relTypes: ["Create", "Update", "Delete"],
                        isOneToOne: false,
                        objectType: "ResourceVersion",
                        parentRelationshipName: "",
                        data,
                        joinData: {
                            fieldName: "id",
                            uniqueFieldName: "routine_version_subroutine_parentRoutineId_subroutineId_unique",
                            childIdFieldName: "subroutineId",
                            parentIdFieldName: "parentRoutineId",
                            parentId: data.id ?? null,
                        },
                        ...rest,
                    }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsVersion({ ...params, objectType: __typename });
            },
        },
        yup: resourceVersionValidation,
    },
    search: {
        defaultSort: ResourceVersionSortBy.DateCompletedDesc,
        sortBy: ResourceVersionSortBy,
        searchFields: {
            codeLanguage: true,
            createdByIdRoot: true,
            createdTimeFrame: true,
            excludeIds: true,
            isCompleteWithRoot: true,
            isCompleteWithRootExcludeOwnedByTeamId: true,
            isCompleteWithRootExcludeOwnedByUserId: true,
            isInternalWithRoot: true,
            isInternalWithRootExcludeOwnedByTeamId: true,
            isInternalWithRootExcludeOwnedByUserId: true,
            isExternalWithRootExcludeOwnedByTeamId: true,
            isExternalWithRootExcludeOwnedByUserId: true,
            isLatest: true,
            maxComplexity: true,
            maxSimplicity: true,
            maxTimesCompleted: true,
            minComplexity: true,
            minSimplicity: true,
            minTimesCompleted: true,
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
            resourceSubType: true,
            resourceSubTypes: true,
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ResourceVersionModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<ResourceVersionModelInfo["DbSelect"]>([["root", "Resource"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<ResourceModelLogic>("Resource").validate().owner(data?.root as ResourceModelInfo["DbModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Resource", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    root: useVisibility("Resource", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Objects you own
                        {
                            root: useVisibility("Resource", "Own", data),
                        },
                        // Public objects
                        {
                            isPrivate: false, // Can't be private
                            root: (useVisibility("ResourceVersion", "Public", data) as { root: object }).root,
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
                            root: useVisibility("Resource", "Own", data),
                        },
                        // Private roots you own
                        {
                            root: {
                                isPrivate: true, // Root is private
                                ...useVisibility("Resource", "Own", data),
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
                            root: useVisibility("Resource", "Own", data),
                        },
                        // Public roots you own
                        {
                            root: {
                                isPrivate: false, // Root is public
                                ...useVisibility("Resource", "Own", data),
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
                        isInternal: false, // Internal routines should never be in search results
                        isPrivate: false, // Root can't be private
                        OR: [
                            // Unowned
                            { ownedByTeam: null, ownedByUser: null },
                            // Owned by public teams
                            { ownedByTeam: useVisibility("Team", "Public", data) },
                            // Owned by public users
                            { ownedByUser: { isPrivate: false, isPrivateResources: false } },
                        ],
                    },
                };
            },
        },
    }),
});
