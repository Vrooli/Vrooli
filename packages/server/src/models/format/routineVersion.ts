import { noNull, shapeHelper } from "../../builders";
import { calculateWeightData, postShapeVersion, translationShapeHelper } from "../../utils";
import { preShapeVersion } from "../../utils/preShapeVersion";
import { Formatter } from "../types";
import { RoutineModel } from "./routine";

const __typename = "RoutineVersion" as const;
export const RoutineVersionFormat: Formatter<ModelRoutineVersionLogic> = {
    gqlRelMap: {
        __typename,
        apiVersion: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        nodes: "Node",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        resourceList: "ResourceList",
        reports: "Report",
        root: "Routine",
        smartContractVersion: "SmartContractVersion",
        suggestedNextByRoutineVersion: "RoutineVersion",
        // you.runs: 'RunRoutine', //TODO
    },
    prismaRelMap: {
        __typename,
        apiVersion: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        reports: "Report",
        smartContractVersion: "SmartContractVersion",
        nodes: "Node",
        nodeLinks: "NodeLink",
        resourceList: "ResourceList",
        root: "Routine",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        runRoutines: "RunRoutine",
        runSteps: "RunRoutineStep",
        suggestedNextByRoutineVersion: "RoutineVersion",
    },
    joinMap: {
        suggestedNextByRoutineVersion: "toRoutineVersion",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        inputsCount: true,
        nodeLinksCount: true,
        nodesCount: true,
        outputsCount: true,
        reportsCount: true,
        suggestedNextByRoutineVersionCount: true,
        translationsCount: true,
    },
}'t affect the calculations. 
// If a deleting version appears in the calculations, an error will be thrown.
const { dataWeights } = await calculateWeightData(
    prisma,
    userData.languages,
    [...createList, ...updateList.map(u => u.data)],
    deleteList,
);
// Convert dataWeights to a map for easy lookup
const dataWeightMap = dataWeights.reduce((acc, curr) => {
    acc[curr.id] = curr;
    return acc;
}, {});
const maps = preShapeVersion({ createList, updateList, objectType: __typename });
return { ...maps, dataWeightMap };
            },
create: async ({ data, ...rest }) => {
    return {
        id: data.id,
        simplicity: rest.preMap[__typename][data.id]?.simplicity ?? 0,
        complexity: rest.preMap[__typename][data.id]?.complexity ?? 0,
        apiCallData: noNull(data.apiCallData),
        isAutomatable: noNull(data.isAutomatable),
        isPrivate: noNull(data.isPrivate),
        isComplete: noNull(data.isComplete),
        smartContractCallData: noNull(data.smartContractCallData),
        versionLabel: data.versionLabel,
        versionNotes: noNull(data.versionNotes),
        ...(await shapeHelper({ relation: "apiVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest })),
        ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childRoutineVersions", data, ...rest })),
        ...(await shapeHelper({ relation: "inputs", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "RoutineVersionInput", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "nodes", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Node", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "nodeLinks", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NodeLink", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "outputs", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "RoutineVersionOutput", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Routine", parentRelationshipName: "versions", data, ...rest })),
        ...(await shapeHelper({ relation: "smartContractVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "SmartContractVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest })),
        // ...(await shapeHelper({ relation: "suggestedNextByRoutineVersion", relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, ...rest })),
        ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
    };
},
    update: async ({ data, ...rest }) => ({
        simplicity: rest.preMap[__typename][data.id]?.simplicity ?? 0,
        complexity: rest.preMap[__typename][data.id]?.complexity ?? 0,
        apiCallData: noNull(data.apiCallData),
        isAutomatable: noNull(data.isAutomatable),
        isPrivate: noNull(data.isPrivate),
        isComplete: noNull(data.isComplete),
        smartContractCallData: noNull(data.smartContractCallData),
        versionLabel: noNull(data.versionLabel),
        versionNotes: noNull(data.versionNotes),
        ...(await shapeHelper({ relation: "apiVersion", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest })),
        ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childRoutineVersions", data, ...rest })),
        ...(await shapeHelper({ relation: "inputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "RoutineVersionInput", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "nodes", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "Node", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "nodeLinks", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "NodeLink", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "outputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "RoutineVersionOutput", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "routineVersion", data, ...rest })),
        ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "Routine", parentRelationshipName: "versions", data, ...rest })),
        ...(await shapeHelper({ relation: "smartContractVersion", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "SmartContractVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest })),
        // ...(await shapeHelper({ relation: "suggestedNextByRoutineVersion", relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, ...rest })),
        ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
    }),
        post: async (params) => {
            await postShapeVersion({ ...params, objectType: __typename });
        },
            searchFields: {
    createdByIdRoot: true,
        createdTimeFrame: true,
            directoryListingsId: true,
                excludeIds: true,
                    isCompleteWithRoot: true,
                        isCompleteWithRootExcludeOwnedByOrganizationId: true,
                            isCompleteWithRootExcludeOwnedByUserId: true,
                                isInternalWithRoot: true,
                                    isInternalWithRootExcludeOwnedByOrganizationId: true,
                                        isInternalWithRootExcludeOwnedByUserId: true,
                                            isExternalWithRootExcludeOwnedByOrganizationId: true,
                                                isExternalWithRootExcludeOwnedByUserId: true,
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
                                                                                                    ownedByOrganizationIdRoot: true,
                                                                                                        ownedByUserIdRoot: true,
                                                                                                            reportId: true,
                                                                                                                rootId: true,
                                                                                                                    tagsRoot: true,
                                                                                                                        translationLanguages: true,
                                                                                                                            updatedTimeFrame: true,
                                                                                                                                searchStringQuery: () => ({
                                                                                                                                    OR: [
                                                                                                                                        "transDescriptionWrapped",
                                                                                                                                        "transNameWrapped",
                                                                                                                                        { root: "tagsWrapped" },
                                                                                                                                        { root: "labelsWrapped" },
                                                                                                                                    ],
                                                                                                                                    permissionsSelect: () => ({
                                                                                                                                        id: true,
                                                                                                                                        isDeleted: true,
                                                                                                                                        isPrivate: true,
                                                                                                                                        root: ["Routine", ["versions"]],
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
                                                                                                                                                root: RoutineModel.validate!.visibility.owner(userId),
                                                                                                                                            }),
                                                                                                                                        };
