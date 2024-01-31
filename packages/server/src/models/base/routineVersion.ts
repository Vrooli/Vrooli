import { MaxObjects, RoutineVersionCreateInput, RoutineVersionSortBy, RoutineVersionUpdateInput, routineVersionValidation } from "@local/shared";
import { ModelMap } from ".";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { noNull } from "../../builders/noNull";
import { selectHelper } from "../../builders/selectHelper";
import { shapeHelper } from "../../builders/shapeHelper";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { PartialGraphQLInfo } from "../../builders/types";
import { PrismaType } from "../../types";
import { bestTranslation, calculateWeightData, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { RoutineVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { RoutineModelInfo, RoutineModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic, RunRoutineModelLogic } from "./types";

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
export const RoutineVersionModel: RoutineVersionModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.routine_version,
    display: () => ({
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
    }),
    format: RoutineVersionFormat,
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
                const combinedInputs = [...Create, ...Update].map(d => d.input);
                combinedInputs.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                await Promise.all(combinedInputs.map(async (input) => { await validateNodePositions(prisma, input, userData.languages); }));
                // Calculate simplicity and complexity of all versions. Since these calculations 
                // can depend on other versions, we need to do them all at once. 
                // We exclude deleting versions to ensure that they don't affect the calculations. 
                // If a deleting version appears in the calculations, an error will be thrown.
                const { dataWeights } = await calculateWeightData(
                    prisma,
                    userData.languages,
                    combinedInputs,
                    Delete.map(d => d.input),
                );
                // Convert dataWeights to a map for easy lookup
                const dataWeightMap = dataWeights.reduce((acc, curr) => {
                    acc[curr.id] = curr;
                    return acc;
                }, {});
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps, dataWeightMap };
            },
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    simplicity: rest.preMap[__typename][data.id]?.simplicity ?? 0,
                    complexity: rest.preMap[__typename][data.id]?.complexity ?? 0,
                    apiCallData: noNull(data.apiCallData),
                    isAutomatable: noNull(data.isAutomatable),
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    smartContractCallData: noNull(data.smartContractCallData),
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    apiVersion: await shapeHelper({ relation: "apiVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ApiVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childRoutineVersions", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create"], isOneToOne: false, objectType: "RoutineVersionInput", parentRelationshipName: "routineVersion", data, ...rest }),
                    nodes: await shapeHelper({ relation: "nodes", relTypes: ["Create"], isOneToOne: false, objectType: "Node", parentRelationshipName: "routineVersion", data, ...rest }),
                    nodeLinks: await shapeHelper({ relation: "nodeLinks", relTypes: ["Create"], isOneToOne: false, objectType: "NodeLink", parentRelationshipName: "routineVersion", data, ...rest }),
                    outputs: await shapeHelper({ relation: "outputs", relTypes: ["Create"], isOneToOne: false, objectType: "RoutineVersionOutput", parentRelationshipName: "routineVersion", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "routineVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Routine", parentRelationshipName: "versions", data, ...rest }),
                    smartContractVersion: await shapeHelper({ relation: "smartContractVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "SmartContractVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                    // ...(await shapeHelper({ relation: "suggestedNextByRoutineVersion", relTypes: ['Connect'], isOneToOne: false,   objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, ...rest })), TODO needs join table
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest }),
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
                apiVersion: await shapeHelper({ relation: "apiVersion", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "ApiVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childRoutineVersions", data, ...rest }),
                inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RoutineVersionInput", parentRelationshipName: "routineVersion", data, ...rest }),
                nodes: await shapeHelper({ relation: "nodes", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "Node", parentRelationshipName: "routineVersion", data, ...rest }),
                nodeLinks: await shapeHelper({ relation: "nodeLinks", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "NodeLink", parentRelationshipName: "routineVersion", data, ...rest }),
                outputs: await shapeHelper({ relation: "outputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RoutineVersionOutput", parentRelationshipName: "routineVersion", data, ...rest }),
                resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "routineVersion", data, ...rest }),
                root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Routine", parentRelationshipName: "versions", data, ...rest }),
                smartContractVersion: await shapeHelper({ relation: "smartContractVersion", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "SmartContractVersion", parentRelationshipName: "calledByRoutineVersions", data, ...rest }),
                // ...(await shapeHelper({ relation: "suggestedNextByRoutineVersion", relTypes: ['Connect', 'Disconnect'], isOneToOne: false,   objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, ...rest })), needs join table
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest }),
            }),
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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, objects, partial, prisma, userData }) => {
                const runs = async () => {
                    if (!userData || !partial.runs) return new Array(objects.length).fill([]);
                    // Find requested fields of runs. Also add routineVersionId, so we 
                    // can associate runs with their routine
                    const runPartial: PartialGraphQLInfo = {
                        ...toPartialGqlInfo(partial.runs as PartialGraphQLInfo, ModelMap.get<RunRoutineModelLogic>("RunRoutine").format.gqlRelMap, userData.languages, true),
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
                root: ModelMap.get<RoutineModelLogic>("Routine").validate().visibility.owner(userId),
            }),
        },
    }),
    validateNodePositions,
});
