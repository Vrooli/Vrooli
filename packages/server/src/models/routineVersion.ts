import { routineVersionValidation } from "@shared/validation";
import { Trigger } from "../events";
import { RoutineVersionSortBy, RoutineVersionSearchInput, RoutineVersionCreateInput, RoutineVersion, RoutineVersionUpdateInput, PrependString, RoutineVersionYou, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { addSupplementalFields, modelToGraphQL, selPad, selectHelper, toPartialGraphQLInfo, noNull, shapeHelper } from "../builders";
import { bestLabel, calculateWeightData, defaultPermissions, oneIsPublic, translationShapeHelper } from "../utils";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { RoutineModel } from "./routine";

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
    // if (uniqueNodes.length < combinedNodes.length) throw new CustomError('NodeDuplicatePosition', {});
    return;
}

const __typename = 'RoutineVersion' as const;
type Permissions = Pick<RoutineVersionYou, 'canComment' | 'canCopy' | 'canDelete' | 'canUpdate' | 'canBookmark' | 'canReport' | 'canRun' | 'canRead' | 'canVote'>;
const suppFields = ['you'] as const;
export const RoutineVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoutineVersionCreateInput,
    GqlUpdate: RoutineVersionUpdateInput,
    GqlModel: RoutineVersion,
    GqlSearch: RoutineVersionSearchInput,
    GqlSort: RoutineVersionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.routine_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.routine_versionUpsertArgs['update'],
    PrismaModel: Prisma.routine_versionGetPayload<SelectWrap<Prisma.routine_versionSelect>>,
    PrismaSelect: Prisma.routine_versionSelect,
    PrismaWhere: Prisma.routine_versionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            apiVersion: 'ApiVersion',
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'Routine',
            inputs: 'RoutineVersionInput',
            nodes: 'Node',
            outputs: 'RoutineVersionOutput',
            pullRequest: 'PullRequest',
            resourceList: 'ResourceList',
            reports: 'Report',
            root: 'Routine',
            smartContractVersion: 'SmartContractVersion',
            suggestedNextByRoutineVersion: 'RoutineVersion',
            // you.runs: 'RunRoutine', //TODO
        },
        prismaRelMap: {
            __typename,
            apiVersion: 'ApiVersion',
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            reports: 'Report',
            smartContractVersion: 'SmartContractVersion',
            nodes: 'Node',
            nodeLinks: 'NodeLink',
            resourceList: 'ResourceList',
            root: 'Routine',
            forks: 'Routine',
            inputs: 'RoutineVersionInput',
            outputs: 'RoutineVersionOutput',
            pullRequest: 'PullRequest',
            runRoutines: 'RunRoutine',
            runSteps: 'RunRoutineStep',
            suggestedNextByRoutineVersion: 'RoutineVersion',
        },
        joinMap: {
            suggestedNextByRoutineVersion: 'toRoutineVersion',
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
                    if (!userData || !partial.runs) return new Array(objects.length).fill([]);
                    // Find requested fields of runs. Also add routineVersionId, so we 
                    // can associate runs with their routine
                    const runPartial: PartialGraphQLInfo = {
                        ...toPartialGraphQLInfo(partial.runs as PartialGraphQLInfo, RunRoutineModel.format.gqlRelMap, userData.languages, true),
                        routineVersionId: true
                    }
                    // Query runs made by user
                    console.log('in routineVersion tographql before runs', selectHelper(partial));
                    let runs: any[] = await prisma.run_routine.findMany({
                        where: {
                            AND: [
                                { routineVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } }
                            ]
                        },
                        ...selectHelper(runPartial)
                    });
                    // Format runs to GraphQL
                    runs = runs.map(r => modelToGraphQL(r, runPartial));
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
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ prisma, userData, data }) => {
                // TODO need to call calculateWeightData with all inputs before calling this function. 
                // Also weightData is grouped by rootRoutineId, not version's ID. So we need to change line below
                // const { weightData } = await calculateWeightData(prisma, userData.languages, [data], []);
                // const { simplicy, complexity } = weightData.find(w => w.id === data.id)!;
                return {
                    id: data.id,
                    // simplicy, TODO also add these to update
                    // complexity,
                    apiCallData: noNull(data.apiCallData),
                    isAutomatable: noNull(data.isAutomatable),
                    isLatest: noNull(data.isLatest),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    smartContractCallData: noNull(data.smartContractCallData),
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    ...(await shapeHelper({ relation: 'apiVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ApiVersion', parentRelationshipName: 'calledByRoutineVersions', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childRoutineVersions', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'inputs', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionInput', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'nodes', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'Node', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'nodeLinks', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NodeLink', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'outputs', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionOutput', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'root', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: true, objectType: 'Routine', parentRelationshipName: 'versions', data, prisma, userData })),
                    ...(await shapeHelper({ relation: 'smartContractVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'SmartContractVersion', parentRelationshipName: 'calledByRoutineVersions', data, prisma, userData })),
                    // ...(await shapeHelper({ relation: 'suggestedNextByRoutineVersion', relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, prisma, userData })),
                    ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
                }
            },
            update: async ({ prisma, userData, data }) => ({
                apiCallData: noNull(data.apiCallData),
                isAutomatable: noNull(data.isAutomatable),
                isLatest: noNull(data.isLatest),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                smartContractCallData: noNull(data.smartContractCallData),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'apiVersion', relTypes: ['Connect', 'Disconnect'], isOneToOne: true, isRequired: false, objectType: 'ApiVersion', parentRelationshipName: 'calledByRoutineVersions', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childRoutineVersions', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'inputs', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionInput', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'nodes', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'Node', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'nodeLinks', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NodeLink', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'outputs', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionOutput', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'routineVersion', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Update'], isOneToOne: true, isRequired: false, objectType: 'Routine', parentRelationshipName: 'versions', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'smartContractVersion', relTypes: ['Connect', 'Disconnect'], isOneToOne: true, isRequired: false, objectType: 'SmartContractVersion', parentRelationshipName: 'calledByRoutineVersions', data, prisma, userData })),
                // ...(await shapeHelper({ relation: 'suggestedNextByRoutineVersion', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersionEndNext', parentRelationshipName: 'fromRoutineVersion', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
            }),
        },
        trigger: {
            onCreated: ({ created, prisma, userData }) => {
                for (const c of created) {
                    Trigger(prisma, userData.languages).createRoutine(userData.id, c.id as string);
                }
            },
            onUpdated: ({ authData, prisma, updated, updateInput, userData }) => {
                // // Initialize transfers, if any
                // asdfasdfasfd
                // // Handle new version triggers, if any versions have been created
                // // Loop through updated items
                // for (let i = 0; i < updated.length; i++) {
                //     const u = updated[i];
                //     const input = updateInput[i];
                //     const permissionsData = authData[u.id];
                //     const { Organization, User } = validator().owner(permissionsData as any);
                //     const owner: { __typename: 'Organization' | 'User', id: string } | null = Organization ?
                //         { __typename: 'Organization', id: Organization.id } :
                //         User ? { __typename: 'User', id: User.id } : null;
                //     const hasOriginalOwner = validator().hasOriginalOwner(permissionsData as any);
                //     const wasPublic = validator().isPublic(permissionsData as any, userData.languages);
                //     const hadCompletedVersion = validator().hasCompleteVersion(permissionsData as any);
                //     const isPublic = input.isPrivate !== undefined ? !input.isPrivate : wasPublic;
                //     const hasCompleteVersion = asdfasdfasdf
                //     // Check if new version was created
                //     if (input.versionLabel) {
                //         Trigger(prisma, userData.languages).objectNewVersion(
                //             userData.id,
                //             'Routine',
                //             u.id,
                //             owner,
                //             hasOriginalOwner,
                //             hadCompletedVersion && wasPublic,
                //             hasCompleteVersion && isPublic
                //         );
                //     }
                // }
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
                'transDescriptionWrapped',
                'transNameWrapped',
                { root: 'tagsWrapped' },
                { root: 'labelsWrapped' },
            ]
        }),
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            RoutineModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => RoutineModel.validate!.owner(data.root as any),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ['Routine', ['versions']],
        }),
        permissionResolvers: defaultPermissions,
        validations: {
            async common({ createMany, deleteMany, languages, prisma, updateMany }) {
                await versionsCheck({
                    createMany,
                    deleteMany,
                    languages,
                    objectType: 'Routine',
                    prisma,
                    updateMany: updateMany as any,
                });
            },
            async create({ createMany, languages, prisma }) {
                createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksBio', languages))
                await Promise.all(createMany.map(async (input) => { await validateNodePositions(prisma, input, languages) }));
            },
            async update({ languages, prisma, updateMany }) {
                updateMany.forEach(({ data }) => lineBreaksCheck(data, ['description'], 'LineBreaksBio', languages));
                await Promise.all(updateMany.map(async (input) => { await validateNodePositions(prisma, input.data, languages) }));
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
                root: RoutineModel.validate!.visibility.owner(userId),
            }),
        },
    },
    validateNodePositions,
})