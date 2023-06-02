import { MaxObjects, RoutineVersionCreateInput, RoutineVersionSortBy, RoutineVersionUpdateInput, routineVersionValidation } from "@local/shared";
import { addSupplementalFields, modelToGql, noNull, selectHelper, shapeHelper, toPartialGqlInfo } from "../../builders";
import { PartialGraphQLInfo } from "../../builders/types";
import { PrismaType } from "../../types";
import { bestTranslation, calculateWeightData, defaultPermissions, getEmbeddableString, postShapeVersion, translationShapeHelper } from "../../utils";
import { preShapeVersion } from "../../utils/preShapeVersion";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { RoutineVersionFormat } from "../format/routineVersion";
import { ModelLogic } from "../types";
import { RoutineModel } from "./routine";
import { RunRoutineModel } from "./runRoutine";
import { RoutineVersionModelLogic } from "./types";

/**
 * Validates node positions
 */
const validateNodePositions = async (
    prisma: PrismaType,
    input: RoutineVersionCreateInput | RoutineVersionUpdateInput,
    languages: string[],
): Promise<void> => {
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
};

const __typename = "RoutineVersion" as const;
const suppFields = ["you"] as const;
export const RoutineVersionModel: ModelLogic<RoutineVersionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.routine_version,
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
                    name: trans.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans.description,
                }, languages[0]);
            },
        },
    },
    format: RoutineVersionFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const { createList, updateList, deleteList, prisma, userData } = params;
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
                // Calculate simplicity and complexity of all versions. Since these calculations 
                // can depend on other versions, we need to do them all at once. 
                // We exclude deleting versions to ensure that they don't affect the calculations. 
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
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, partial, prisma, userData }) => {
                const runs = async () => {
                    if (!userData || !partial.runs) return new Array(objects.length).fill([]);
                    // Find requested fields of runs. Also add routineVersionId, so we 
                    // can associate runs with their routine
                    const runPartial: PartialGraphQLInfo = {
                        ...toPartialGqlInfo(partial.runs as PartialGraphQLInfo, RunRoutineModel.format.gqlRelMap, userData.languages, true),
                        routineVersionId: true,
                    };
                    // Query runs made by user
                    let runs: any[] = await prisma.run_routine.findMany({
                        where: {
                            AND: [
                                { routineVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } },
                            ],
                        },
                        ...selectHelper(runPartial),
                    });
                    // Format runs to GraphQL
                    runs = runs.map(r => modelToGql(r, runPartial));
                    // Add supplemental fields
                    runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                    // Split runs by id
                    const routineRuns = ids.map((id) => runs.filter(r => r.routineVersionId === id));
                    return routineRuns;
                };
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        runs: await runs(),
                    },
                };
            },
        },
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            RoutineModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => RoutineModel.validate!.owner(data.root as any, userId),
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
                root: RoutineModel.validate!.visibility.owner(userId),
            }),
        },
    },
    validateNodePositions,
});
