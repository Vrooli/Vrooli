import { routineVersionValidation } from "@shared/validation";
import { Trigger } from "../events";
import { RoutineVersionSortBy, RoutineVersionSearchInput, RoutineVersionCreateInput, RoutineVersion, RoutineVersionUpdateInput, PrependString, RoutineVersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { addSupplementalFields, modelToGraphQL, padSelect, selectHelper, toPartialGraphQLInfo } from "../builders";
import { bestLabel, defaultPermissions, oneIsPublic } from "../utils";
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
            create: async ({ data, prisma, userData }) => {
                return {} as any;
            },
            update: async ({ data, prisma, userData, where }) => {
                return {} as any;
            },
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
            createdById: true,
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
            ownedByOrganizationId: true,
            ownedByUserId: true,
            reportId: true,
            rootId: true,
            tags: true,
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
        maxObjects: 1000000,
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
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ]
            },
            public: {
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