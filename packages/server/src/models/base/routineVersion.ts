import { MaxObjects, RoutineVersionCreateInput, RoutineVersionSortBy, RoutineVersionUpdateInput, getTranslation, routineVersionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { SubroutineWeightData, calculateWeightData, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { PreShapeVersionResult, afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { RoutineVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { RoutineModelInfo, RoutineModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic } from "./types";

type RoutineVersionPre = PreShapeVersionResult & {
    /** Map of routine version ID to graph complexity metrics */
    weightMap: Record<string, SubroutineWeightData & { id: string; }>;
};

/**
 * Validates node positions
 */
async function validateNodePositions(
    input: RoutineVersionCreateInput | RoutineVersionUpdateInput,
    languages: string[],
): Promise<void> {
    // // Check that node columnIndexes and rowIndexes are valid TODO query existing data to do this
    // let combinedNodes = [];
    // if (input.nodesCreate) combinedNodes.push(...input.nodesCreate);
    // if ((input as RoutineUpdateInput).nodesUpdate) combinedNodes.push(...(input as any).nodesUpdate);
    // if ((input as RoutineUpdateInput).nodesDelete) combinedNodes = combinedNodes.filter(node => !(input as any).nodesDelete.includes(node.id));
    // // Remove nodes that have duplicate rowIndexes and columnIndexes
    // console.log('unique nodes check', JSON.stringify(combinedNodes));
    // const uniqueNodes = uniqBy(combinedNodes, (n) => `${n.rowIndex}-${n.columnIndex}`);
    // if (uniqueNodes.length < combinedNodes.length) throw new CustomError("NodeDuplicatePosition", {});
    return;
}

const __typename = "RoutineVersion" as const;
export const RoutineVersionModel: RoutineVersionModelLogic = ({
    __typename,
    dbTable: "routine_version",
    dbTranslationTable: "routine_version_translation",
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
                }, languages[0]);
            },
        },
    }),
    format: RoutineVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<RoutineVersionPre> => {
                const { Create, Update, Delete, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    Update,
                    userData,
                });
                const combinedInputs = [...Create, ...Update].map(d => d.input);
                combinedInputs.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                await Promise.all(combinedInputs.map(async (input) => { await validateNodePositions(input, userData.languages); }));
                // Calculate simplicity and complexity of all versions. Since these calculations 
                // can depend on other versions, we need to do them all at once. 
                // We exclude deleting versions to ensure that they don't affect the calculations. 
                // If a deleting version appears in the calculations, an error will be thrown.
                const { dataWeights } = await calculateWeightData(
                    userData.languages,
                    combinedInputs,
                    Delete.map(d => d.input),
                );
                // Convert dataWeights to a map for easy lookup
                const weightMap: RoutineVersionPre["weightMap"] = dataWeights.reduce((acc, curr) => {
                    acc[curr.id] = curr;
                    return acc;
                }, {});
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps, weightMap };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as RoutineVersionPre;
                return {
                    id: data.id,
                    simplicity: preData.weightMap[data.id]?.simplicity ?? 0,
                    complexity: preData.weightMap[data.id]?.complexity ?? 0,
                    configCallData: noNull(data.configCallData),
                    configFormInput: noNull(data.configFormInput),
                    configFormOutput: noNull(data.configFormOutput),
                    isAutomatable: noNull(data.isAutomatable),
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    routineType: data.routineType,
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    apiVersion: await shapeHelper({ relation: "apiVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ApiVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                    codeVersion: await shapeHelper({ relation: "codeVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "CodeVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childRoutineVersions", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create"], isOneToOne: false, objectType: "RoutineVersionInput", parentRelationshipName: "routineVersion", data, ...rest }),
                    nodes: await shapeHelper({ relation: "nodes", relTypes: ["Create"], isOneToOne: false, objectType: "Node", parentRelationshipName: "routineVersion", data, ...rest }),
                    nodeLinks: await shapeHelper({ relation: "nodeLinks", relTypes: ["Create"], isOneToOne: false, objectType: "NodeLink", parentRelationshipName: "routineVersion", data, ...rest }),
                    outputs: await shapeHelper({ relation: "outputs", relTypes: ["Create"], isOneToOne: false, objectType: "RoutineVersionOutput", parentRelationshipName: "routineVersion", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "routineVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Routine", parentRelationshipName: "versions", data, ...rest }),
                    // ...(await shapeHelper({ relation: "suggestedNextByRoutineVersion", relTypes: ['Connect'], isOneToOne: false,   objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, ...rest })), TODO needs join table
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as RoutineVersionPre;
                return {
                    simplicity: preData.weightMap[data.id]?.simplicity ?? 0,
                    complexity: preData.weightMap[data.id]?.complexity ?? 0,
                    configCallData: noNull(data.configCallData),
                    configFormInput: noNull(data.configFormInput),
                    configFormOutput: noNull(data.configFormOutput),
                    isAutomatable: noNull(data.isAutomatable),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    apiVersion: await shapeHelper({ relation: "apiVersion", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "ApiVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                    codeVersion: await shapeHelper({ relation: "codeVersion", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "CodeVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childRoutineVersions", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RoutineVersionInput", parentRelationshipName: "routineVersion", data, ...rest }),
                    nodes: await shapeHelper({ relation: "nodes", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "Node", parentRelationshipName: "routineVersion", data, ...rest }),
                    nodeLinks: await shapeHelper({ relation: "nodeLinks", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "NodeLink", parentRelationshipName: "routineVersion", data, ...rest }),
                    outputs: await shapeHelper({ relation: "outputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RoutineVersionOutput", parentRelationshipName: "routineVersion", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "routineVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Routine", parentRelationshipName: "versions", data, ...rest }),
                    // ...(await shapeHelper({ relation: "suggestedNextByRoutineVersion", relTypes: ['Connect', 'Disconnect'], isOneToOne: false,   objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, ...rest })), needs join table
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsVersion({ ...params, objectType: __typename });
            },
        },
        yup: routineVersionValidation,
    },
    search: {
        defaultSort: RoutineVersionSortBy.DateCompletedDesc,
        sortBy: RoutineVersionSortBy,
        searchFields: {
            codeVersionId: true,
            createdByIdRoot: true,
            createdTimeFrame: true,
            directoryListingsId: true,
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
            routineType: true,
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<RoutineVersionModelInfo["GqlPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<RoutineVersionModelInfo["PrismaSelect"]>([["root", "Routine"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<RoutineModelLogic>("Routine").validate().owner(data?.root as RoutineModelInfo["PrismaModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Routine", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    root: useVisibility("Routine", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Objects you own
                        {
                            root: useVisibility("Routine", "Own", data),
                        },
                        // Public objects
                        {
                            isPrivate: false, // Can't be private
                            root: (useVisibility("RoutineVersion", "Public", data) as { root: object }).root,
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
                            root: useVisibility("Routine", "Own", data),
                        },
                        // Private roots you own
                        {
                            root: {
                                isPrivate: true, // Root is private
                                ...useVisibility("Routine", "Own", data),
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
                            root: useVisibility("Routine", "Own", data),
                        },
                        // Public roots you own
                        {
                            root: {
                                isPrivate: false, // Root is public
                                ...useVisibility("Routine", "Own", data),
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
                            { ownedByUser: { isPrivate: false, isPrivateRoutines: false } },
                        ],
                    },
                };
            },
        },
    }),
    validateNodePositions,
});
