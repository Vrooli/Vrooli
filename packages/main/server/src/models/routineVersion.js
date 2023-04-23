import { MaxObjects, RoutineVersionSortBy } from "@local/consts";
import { routineVersionValidation } from "@local/validation";
import { addSupplementalFields, modelToGql, noNull, selectHelper, shapeHelper, toPartialGqlInfo } from "../builders";
import { bestLabel, calculateWeightData, defaultPermissions, postShapeVersion, translationShapeHelper } from "../utils";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { RoutineModel } from "./routine";
import { RunRoutineModel } from "./runRoutine";
const validateNodePositions = async (prisma, input, languages) => {
    return;
};
const __typename = "RoutineVersion";
const suppFields = ["you"];
export const RoutineVersionModel = ({
    __typename,
    delegate: (prisma) => prisma.routine_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
    },
    format: {
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
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, partial, prisma, userData }) => {
                const runs = async () => {
                    if (!userData || !partial.runs)
                        return new Array(objects.length).fill([]);
                    const runPartial = {
                        ...toPartialGqlInfo(partial.runs, RunRoutineModel.format.gqlRelMap, userData.languages, true),
                        routineVersionId: true,
                    };
                    let runs = await prisma.run_routine.findMany({
                        where: {
                            AND: [
                                { routineVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } },
                            ],
                        },
                        ...selectHelper(runPartial),
                    });
                    runs = runs.map(r => modelToGql(r, runPartial));
                    runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                    const routineRuns = ids.map((id) => runs.filter(r => r.routineVersionId === id));
                    return routineRuns;
                };
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                        runs: await runs(),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList, deleteList, prisma, userData }) => {
                await versionsCheck({
                    createList,
                    deleteList,
                    objectType: __typename,
                    prisma,
                    updateList,
                    userData,
                });
                const combined = [...createList, ...updateList.map(({ data }) => data)];
                combined.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                await Promise.all(combined.map(async (input) => { await validateNodePositions(prisma, input, userData.languages); }));
                const { dataWeights } = await calculateWeightData(prisma, userData.languages, [...createList, ...updateList.map(u => u.data)], deleteList);
                const dataWeightMap = dataWeights.reduce((acc, curr) => {
                    acc[curr.id] = curr;
                    return acc;
                }, {});
                return dataWeightMap;
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
                    ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
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
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
            },
        },
        yup: routineVersionValidation,
    },
    search: {
        defaultSort: RoutineVersionSortBy.DateCompletedDesc,
        sortBy: RoutineVersionSortBy,
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
            ],
        }),
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            RoutineModel.validate.isPublic(data.root, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => RoutineModel.validate.owner(data.root, userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Routine", ["versions"]],
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
                root: RoutineModel.validate.visibility.owner(userId),
            }),
        },
    },
    validateNodePositions,
});
//# sourceMappingURL=routineVersion.js.map