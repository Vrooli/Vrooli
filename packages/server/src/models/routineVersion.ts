import { routinesCreate, routinesUpdate } from "@shared/validation";
import { ResourceListUsedFor } from "@shared/consts";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { CustomError, Trigger } from "../events";
import { Routine, RoutinePermission, RoutineSearchInput, RoutineCreateInput, RoutineUpdateInput, NodeCreateInput, NodeUpdateInput, NodeRoutineListItem, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListUpdateInput, RoutineSortBy, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Validator, Mutater, Displayer, Duplicator } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { relBuilderHelper } from "../actions";
import { getSingleTypePermissions } from "../validators";
import { RunModel } from "./run";
import { PartialGraphQLInfo } from "../builders/types";
import { addSupplementalFields, combineQueries, exceptionsBuilder, modelToGraphQL, padSelect, permissionsSelectHelper, selectHelper, toPartialGraphQLInfo, visibilityBuilder } from "../builders";
import { bestLabel, oneIsPublic, tagRelationshipBuilder, translationRelationshipBuilder } from "../utils";

type NodeWeightData = {
    simplicity: number,
    complexity: number,
    optionalInputs: number,
    allInputs: number,
}

type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsRoutine' | 'runs' | 'versions';
const formatter = (): Formatter<Routine, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Routine',
        comments: 'Comment',
        creator: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        forks: 'Routine',
        inputs: 'InputItem',
        nodes: 'Node',
        outputs: 'OutputItem',
        owner: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        parent: 'Routine',
        project: 'Project',
        reports: 'Report',
        resourceLists: 'ResourceList',
        starredBy: 'User',
        tags: 'Tag',
    },
    rootFields: ['hasCompleteVersion', 'isDeleted', 'isInternal', 'isPrivate', 'votes', 'stars', 'views', 'permissions'],
    joinMap: { tags: 'tag', starredBy: 'user' },
    countMap: { commentsCount: 'comments', nodesCount: 'nodes', reportsCount: 'reports' },
    supplemental: {
        graphqlFields: ['isUpvoted', 'isStarred', 'isViewed', 'permissionsRoutine', 'runs', 'versions'],
        toGraphQL: ({ ids, objects, partial, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'Routine')],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, 'Routine')],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, 'Routine')],
            ['permissionsRoutine', async () => await getSingleTypePermissions('Routine', ids, prisma, userData)],
            ['runs', async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // Find requested fields of runs. Also add routineId, so we 
                // can associate runs with their routine
                const runPartial: PartialGraphQLInfo = {
                    ...toPartialGraphQLInfo(partial.runs as PartialGraphQLInfo, RunModel.format.relationshipMap, userData.languages, true),
                    routineVersionId: true
                }
                // Query runs made by user
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
                const routineRuns = ids.map((id) => runs.filter(r => r.routineId === id));
                return routineRuns;
            }],
            ['versions', async () => {
                const groupData = await prisma.routine.findMany({
                    where: { id: { in: ids } },
                    select: { versions: { select: { id: true, versionLabel: true } } }
                });
                return groupData.map(g => g.versions);
            }],
        ],
    },
})

const searcher = (): Searcher<
    RoutineSearchInput,
    RoutineSortBy,
    Prisma.routine_versionOrderByWithRelationInput,
    Prisma.routine_versionWhereInput
> => ({
    defaultSort: RoutineSortBy.VotesDesc,
    sortMap: {
        CommentsAsc: { comments: { _count: 'asc' } },
        CommentsDesc: { comments: { _count: 'desc' } },
        ForksAsc: { forks: { _count: 'asc' } },
        ForksDesc: { forks: { _count: 'desc' } },
        DateCompletedAsc: { completedAt: 'asc' },
        DateCompletedDesc: { completedAt: 'desc' },
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        StarsAsc: { root: { stars: 'asc' } },
        StarsDesc: { root: { stars: 'desc' } },
        VotesAsc: { root: { votes: 'asc' } },
        VotesDesc: { root: { votes: 'desc' } },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
            { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
            { tags: { some: { tag: { tag: { ...insensitive } } } } },
        ]
    }),
    customQueries(input, userData) {
        const isComplete = exceptionsBuilder({
            canQuery: ['createdByOrganization', 'createdByUser', 'organization.id', 'project.id', 'user.id'],
            exceptionField: 'isCompleteExceptions',
            input,
            mainField: 'isComplete',
        })
        const isInternal = exceptionsBuilder({
            canQuery: ['createdByOrganization', 'createdByUser', 'organization.id', 'project.id', 'user.id'],
            defaultValue: false,
            exceptionField: 'isInternalExceptions',
            input,
            mainField: 'isInternal',
        })
        console.log('before routine customqueries combinequeries', JSON.stringify(isComplete), '\n', JSON.stringify(isInternal), '\n');
        return combineQueries([
            isComplete,
            isInternal,
            visibilityBuilder({ objectType: 'Routine', userData, visibility: input.visibility }),
            (input.excludeIds !== undefined ? { NOT: { id: { in: input.excludeIds } } } : {}),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minComplexity !== undefined ? { complexity: { gte: input.minComplexity } } : {}),
            (input.maxComplexity !== undefined ? { complexity: { lte: input.maxComplexity } } : {}),
            (input.minSimplicity !== undefined ? { simplicity: { gte: input.minSimplicity } } : {}),
            (input.maxSimplicity !== undefined ? { simplicity: { lte: input.maxSimplicity } } : {}),
            (input.maxTimesCompleted !== undefined ? { timesCompleted: { lte: input.maxTimesCompleted } } : {}),
            (input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.minTimesCompleted !== undefined ? { timesCompleted: { gte: input.minTimesCompleted } } : {}),
            (input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            (input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            (input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            (input.userId !== undefined ? { userId: input.userId } : {}),
            (input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            (input.projectId !== undefined ? { projectId: input.projectId } : {}),
            (input.parentId !== undefined ? { parentId: input.parentId } : {}),
            (input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            (input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        ])
    },
})

const validator = (): Validator<
    RoutineCreateInput,
    RoutineUpdateInput,
    Prisma.routineGetPayload<{ select: { [K in keyof Required<Prisma.routineSelect>]: true } }>,
    RoutinePermission,
    Prisma.routineSelect,
    Prisma.routineWhereInput,
    true,
    true
> => ({
    validateMap: {
        __typename: 'Routine',
        parent: 'Routine',
        createdBy: 'User',
        ownedByOrganization: 'Organization',
        ownedByUser: 'User',
        versions: {
            select: {
                forks: 'Routine',
                // api: 'Api',
                // smartContract: 'SmartContract',
                // directoryListings: 'ProjectDirectory',
            }
        }
    },
    isTransferable: true,
    hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
    maxObjects: {
        User: {
            private: {
                noPremium: 25,
                premium: 250,
            },
            public: {
                noPremium: 100,
                premium: 2000,
            },
        },
        Organization: {
            private: {
                noPremium: 25,
                premium: 250,
            },
            public: {
                noPremium: 100,
                premium: 2000,
            },
        },
    },
    hasCompletedVersion: (data) => data.hasCompleteVersion === true,
    permissionsSelect: (...params) => ({
        id: true,
        hasCompleteVersion: true,
        isDeleted: true,
        isPrivate: true,
        isInternal: true,
        permissions: true,
        createdBy: padSelect({ id: true }),
        ...permissionsSelectHelper([
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], ...params),
        versions: {
            select: {
                id: true,
                isComplete: true,
                isPrivate: true,
                isDeleted: true,
                versionIndex: true,
            }
        }
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ([
        ['canComment', async () => !isDeleted && (isAdmin || isPublic)],
        ['canDelete', async () => isAdmin && !isDeleted],
        ['canEdit', async () => isAdmin && !isDeleted],
        ['canReport', async () => !isAdmin && !isDeleted && isPublic],
        ['canRun', async () => !isDeleted && (isAdmin || isPublic)],
        ['canStar', async () => !isDeleted && (isAdmin || isPublic)],
        ['canView', async () => !isDeleted && (isAdmin || isPublic)],
        ['canVote', async () => !isDeleted && (isAdmin || isPublic)],
    ]),
    owner: (data) => ({
        Organization: data.ownedByOrganization,
        User: data.ownedByUser,
    }),
    isDeleted: (data) => data.isDeleted,// || latest(data.versions)?.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false &&
        data.isDeleted === false &&
        data.isInternal === false &&
        //latest(data.versions)?.isPrivate === false &&
        //latest(data.versions)?.isDeleted === false &&
        oneIsPublic<Prisma.routineSelect>(data, [
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], languages),
    visibility: {
        private: {
            isPrivate: true,
            // OR: [
            //     { isPrivate: true },
            //     { root: { isPrivate: true } },
            // ]
        },
        public: {
            isPrivate: false,
            // AND: [
            //     { isPrivate: false },
            //     { root: { isPrivate: false } },
            // ]
        },
        owner: (userId) => ({
            OR: [
                { ownedByUser: { id: userId } },
                { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            ]
            // root: {
            //     OR: [
            //         { ownedByUser: { id: userId } },
            //         { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            //     ]
            // }
        }),
    },
    // if (createMany) {
    //     createMany.forEach(input => this.validateNodePositions(input));
    // }
    // if (updateMany) {
    //     // Query version numbers and isCompletes of existing routines. 
    //     // Can only update if version number is greater, or if version number is the same and isComplete is false
    //     //TODO
    //     updateMany.forEach(input => this.validateNodePositions(input.data));
    // }

    // Also check profanity on input/output's name
})

/**
 * Calculates the shortest AND longest weighted path on a directed cyclic graph. (loops are actually not the cyclic part, but redirects)
 * A routine with no nodes has a complexity equal to the number of its inputs.
 * Each decision the user makes (i.e. multiple edges coming out of a node) has a weight of 1.
 * Each node has a weight that is the summation of its contained subroutines.
 * @param nodes A map of node IDs to their weight (simplicity/complexity)
 * @param edges The edges of the graph, with each object containing a fromId and toId
 * @param languages Preferred languages for error messages
 * @returns [shortestPath, longestPath] The shortest and longest weighted distance
 */
const calculateShortestLongestWeightedPath = (
    nodes: { [id: string]: NodeWeightData },
    edges: { fromId: string, toId: string }[],
    languages: string[]
): [number, number] => {
    // First, check that all edges point to valid nodes. 
    // If this isn't the case, this algorithm could run into an error
    for (const edge of edges) {
        if (!nodes[edge.toId] || !nodes[edge.fromId]) {
            throw new CustomError('0237', 'UnlinkedNodes', languages, { failedEdge: edge })
        }
    }
    // If no nodes or edges, return 1
    if (Object.keys(nodes).length === 0 || edges.length === 0) return [1, 1];
    // Create a dictionary of edges, where the key is a node ID and the value is an array of edges that END at that node
    const edgesByNode: { [id: string]: { fromId: string, toId: string }[] } = {};
    edges.forEach(edge => {
        edgesByNode[edge.toId] = edgesByNode[edge.toId] ? edgesByNode[edge.toId].concat(edge) : [edge];
    });
    /**
     * Calculates the shortest and longest weighted distance
     * @param currentNodeId The current node ID
     * @param visitedEdges The edges that have been visited so far
     * @param currShortest The current shortest distance
     * @param currLongest The current longest distance
     * @returns [shortest, longest] The shortest and longest distance. -1 if doesn't 
     * end with a start node (i.e. caught in a loop)
     */
    const getShortLong = (currentNodeId: string, visitedEdges: { fromId: string, toId: string }[], currShortest: number, currLongest: number): [number, number] => {
        const fromEdges = edgesByNode[currentNodeId];
        // If no from edges, must be start node. Return currShortest and currLongest unchanged
        if (!fromEdges || fromEdges.length === 0) return [currShortest, currLongest];
        // If edges but all have been visited, must be a loop. Return -1
        if (fromEdges.every(edge => visitedEdges.some(visitedEdge => visitedEdge.fromId === edge.fromId && visitedEdge.toId === edge.toId))) return [-1, -1];
        // Otherwise, calculate the shortest and longest distance
        let edgeShorts: number[] = [];
        let edgeLongs: number[] = [];
        for (const edge of fromEdges) {
            // Find the weight of the edge from the node's complexity. Add one if there are multiple edges,
            // since the user has to make a decision
            let weight = nodes[edge.fromId];
            if (fromEdges.length > 1) weight = { ...weight, complexity: weight.complexity + 1, simplicity: weight.simplicity + 1 };
            // Add edge to visited edges
            const newVisitedEdges = visitedEdges.concat([edge]);
            // Recurse on the next node 
            const [shortest, longest] = getShortLong(
                edge.fromId,
                newVisitedEdges,
                currShortest + weight.simplicity + weight.optionalInputs,
                currLongest + weight.complexity + weight.allInputs);
            // If shortest is not -1, add to edgeShorts
            if (shortest !== -1) edgeShorts.push(shortest);
            // If longest is not -1, add to edgeLongs
            if (longest !== -1) edgeLongs.push(longest);
        }
        // Calculate the shortest and longest distance
        const shortest = edgeShorts.length > 0 ? Math.min(...edgeShorts) : -1;
        const longest = edgeLongs.length > 0 ? Math.max(...edgeLongs) : -1;
        return [shortest, longest];
    }
    // Find all of the end nodes, by finding all nodes without any outgoing edges
    const endNodes = Object.keys(nodes).filter(nodeId => !edges.find(e => e.fromId === nodeId));
    // Calculate the shortest and longest for each end node
    const distances: [number, number][] = endNodes.map(nodeId => getShortLong(nodeId, [], 0, 0));
    // Return shortest short and longest long
    return [
        Math.min(...distances.map(d => d[0])),
        Math.max(...distances.map(d => d[1]))
    ]
}

// const routineDuplicater = (): Duplicator<Prisma.routine_versionSelect, Prisma.routine_versionUpsertArgs['create']> => ({
//     select: {
//         id: true,
//         apiCallData: true,
//         complexity: true,
//         isAutomatable: true,
//         simplicity: true,
//         root: {
//             select: {
//                 isInternal: true,
//                 tags: {
//                     select: {
//                         id: true,
//                     }
//                 },
//             }
//         },
//         // Only select top-level nodes
//         nodes: {
//             select: {
//                 id: true,
//                 columnIndex: true,
//                 rowIndex: true,
//                 type: true,
//                 nodeEnd: {
//                     select: {
//                         wasSuccessful: true
//                     }
//                 },
//                 loop: {
//                     select: {
//                         loops: true,
//                         maxLoops: true,
//                         operation: true,
//                         whiles: {
//                             select: {
//                                 condition: true,
//                                 translations: {
//                                     select: {
//                                         description: true,
//                                         title: true,
//                                         language: true,
//                                     }
//                                 }
//                             }
//                         }
//                     }
//                 },
//                 nodeRoutineList: {
//                     select: {
//                         isOrdered: true,
//                         isOptional: true,
//                         routines: {
//                             select: {
//                                 id: true,
//                                 index: true,
//                                 isOptional: true,
//                                 routineVersion: {
//                                     select: {
//                                         id: true,
//                                         root: {
//                                             select: {
//                                                 isInternal: true,
//                                             }
//                                         }
//                                     }
//                                 },
//                                 translations: {
//                                     select: {
//                                         description: true,
//                                         title: true,
//                                         language: true,
//                                     }
//                                 }
//                             }
//                         }
//                     }
//                 },
//                 translations: {
//                     select: {
//                         description: true,
//                         title: true,
//                         language: true,
//                     }
//                 }
//             }
//         },
//         nodeLinks: {
//             select: {
//                 fromId: true,
//                 toId: true,
//                 whens: {
//                     select: {
//                         condition: true,
//                         translations: {
//                             select: {
//                                 description: true,
//                                 title: true,
//                                 language: true,
//                             }
//                         }
//                     }
//                 }
//             }
//         },
//         resourceList: {
//             select: {
//                 index: true,
//                 resources: {
//                     select: {
//                         index: true,
//                         link: true,
//                         usedFor: true,
//                         translations: {
//                             select: {
//                                 description: true,
//                                 title: true,
//                                 language: true,
//                             }
//                         }
//                     }
//                 },
//                 translations: {
//                     select: {
//                         description: true,
//                         title: true,
//                         language: true,
//                     }
//                 }
//             }
//         },
//         inputs: {
//             select: {
//                 isRequired: true,
//                 name: true,
//                 standardVersionId: true,
//                 translations: {
//                     select: {
//                         description: true,
//                         language: true,
//                     }
//                 }
//             }
//         },
//         outputs: {
//             select: {
//                 name: true,
//                 standardVersionId: true,
//                 translations: {
//                     select: {
//                         description: true,
//                         language: true,
//                     }
//                 }
//             }
//         },
//         translations: {
//             select: {
//                 description: true,
//                 instructions: true,
//                 title: true,
//                 language: true,
//             }
//         }
//     },
//     // validateSelect: {
//     //     nodes: {
//     //         select: {
//     //             nodeRoutineList: {
//     //                 select: {
//     //                     routines: {
//     //                         select: {
//     //                             routineVersion: 'Routine',
//     //                         }
//     //                     }
//     //                 }
//     //             }
//     //         }
//     //     }
//     // }
// })

/**
 * Calculates the maximum and minimum complexity of a routine based on the number of steps. 
 * Simplicity is a the minimum number of inputs and decisions required to complete the routine, while 
 * complexity is the maximum.
 * @param prisma The prisma client
 * @param languages Preferred languages for error messages
 * @param data The routine data, either a create or update
 * @param versionId If existing data, its version ID
 * @returns [simplicity, complexity] Numbers representing the shorted and longest weighted paths
 */
const calculateComplexity = async (
    prisma: PrismaType,
    languages: string[],
    data: RoutineCreateInput | RoutineUpdateInput,
    versionId?: string | null
): Promise<[number, number]> => {
    // If the routine is being updated, Find the complexity of existing subroutines
    let existingRoutine;
    if (versionId) {
        existingRoutine = await prisma.routine_version.findUnique({
            where: { id: versionId },
            select: {
                nodeLinks: { select: { id: true, fromId: true, toId: true } },
                nodes: {
                    select: {
                        id: true, // Needed to associate with links
                        nodeRoutineList: {
                            select: {
                                routines: {
                                    select: {
                                        routineVersion: { select: { id: true, complexity: true, simplicity: true } }
                                    }
                                },
                            }
                        },
                    }
                },
            }
        })
    }
    // Calculate the list of links after mutations are applied
    let nodeLinks: any[] = existingRoutine?.nodeLinks || [];
    if (data.nodeLinksCreate) nodeLinks = nodeLinks.concat(data.nodeLinksCreate);
    if ((data as RoutineUpdateInput).nodeLinksUpdate) {
        nodeLinks = nodeLinks.map(link => {
            const updatedLink = (data as RoutineUpdateInput).nodeLinksUpdate?.find(updatedLink => link.id && updatedLink.id === link.id);
            return updatedLink ? { ...link, ...updatedLink } : link;
        })
    }
    if ((data as RoutineUpdateInput).nodeLinksDelete) {
        nodeLinks = nodeLinks.filter(link => !(data as RoutineUpdateInput).nodeLinksDelete?.find(deletedLink => deletedLink === link.id));
    }
    // Calculate the list of nodes after mutations are applied
    let nodes: any[] = existingRoutine?.nodes || [];
    if (data.nodesCreate) nodes = nodes.concat(data.nodesCreate);
    if ((data as RoutineUpdateInput).nodesUpdate) {
        nodes = nodes.map(node => {
            const updatedNode = (data as RoutineUpdateInput).nodesUpdate?.find(updatedNode => node.id && updatedNode.id === node.id);
            return updatedNode ? { ...node, ...updatedNode } : node;
        })
    }
    if ((data as RoutineUpdateInput).nodesDelete) {
        nodes = nodes.filter(node => !(data as RoutineUpdateInput).nodesDelete?.find(deletedNode => deletedNode === node.id));
    }
    // Initialize node dictionary to map node IDs to their subroutine IDs
    const subroutineIdsByNode: { [id: string]: string[] } = {};
    // Find the ID of every subroutine
    const subroutineIds: string[] = nodes.map((node: any | NodeCreateInput | NodeUpdateInput) => {
        // Calculate the list of subroutines after mutations are applied
        let ids: string[] = node.nodeRoutineList?.routines?.map((item: NodeRoutineListItem) => item.routineVersion.id) ?? [];
        if ((data as NodeCreateInput).nodeRoutineListCreate) {
            const listCreate = (data as NodeCreateInput).nodeRoutineListCreate as NodeRoutineListCreateInput;
            // Handle creates
            ids = ids.concat(listCreate.routinesCreate?.map((item: NodeRoutineListItemCreateInput) => item.routineVersionConnect) ?? []);
        }
        else if ((data as NodeUpdateInput).nodeRoutineListUpdate) {
            const listUpdate = (data as NodeUpdateInput).nodeRoutineListUpdate as NodeRoutineListUpdateInput;
            // Handle creates
            ids = ids.concat(listUpdate.routinesCreate?.map((item: NodeRoutineListItemCreateInput) => item.routineVersionConnect) ?? []);
            // Handle deletes. No need to handle updates, as routine items cannot switch their routine associations
            if (listUpdate.routinesDelete) {
                ids = ids.filter(id => !listUpdate.routinesDelete?.find(deletedId => deletedId === id));
            }
        }
        subroutineIdsByNode[node.id] = ids;
        return ids
    }).flat();
    // Query every subroutine's complexity, simplicity, and number of inputs
    const subroutineWeightData = await prisma.routine_version.findMany({
        where: { id: { in: subroutineIds } },
        select: {
            id: true,
            complexity: true,
            simplicity: true,
            inputs: {
                select: {
                    isRequired: true
                }
            }
        }
    })
    // Convert compexity/simplicity to a map for easy lookup
    let subroutineWeightDataDict: { [routineId: string]: NodeWeightData } = {};
    for (const sub of subroutineWeightData) {
        subroutineWeightDataDict[sub.id] = {
            complexity: sub.complexity,
            simplicity: sub.simplicity,
            optionalInputs: sub.inputs.filter(input => !input.isRequired).length,
            allInputs: sub.inputs.length,
        };
    }
    // Calculate the complexity/simplicity of each node. If node has no subroutines, its complexity is 0 (e.g. start node, end node)
    let nodeWeightDataDict: { [nodeId: string]: NodeWeightData } = {};
    for (const node of nodes) {
        nodeWeightDataDict[node.id] = {
            complexity: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].complexity, 0) || 0,
            simplicity: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].simplicity, 0) || 0,
            optionalInputs: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].optionalInputs, 0) || 0,
            allInputs: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].allInputs, 0) || 0,
        }
    }
    // Using the node links, determine the most complex path through the routine
    const [shortest, longest] = calculateShortestLongestWeightedPath(nodeWeightDataDict, nodeLinks, languages);
    // return with +1, so that nesting routines has a small factor in determining weight
    return [shortest + 1, longest + 1];
}

/**
 * Validates node positions
 */
const validateNodePositions = (input: RoutineCreateInput | RoutineUpdateInput): void => {
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

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: RoutineCreateInput | RoutineUpdateInput, isAdd: boolean) => {
    return {
        root: {
            isPrivate: data.isPrivate ?? undefined,
            hasCompleteVersion: (data.isComplete === true) ? true : (data.isComplete === false) ? false : undefined,
            completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
            project: data.projectId ? { connect: { id: data.projectId } } : undefined,
            tags: await tagRelationshipBuilder(prisma, userData, data, 'Routine', isAdd),
            permissions: JSON.stringify({}),
        },
        version: {
            isAutomatable: data.isAutomatable ?? undefined,
            isComplete: data.isComplete ?? undefined,
            isInternal: data.isInternal ?? undefined,
            versionLabel: data.versionLabel ?? undefined,
            resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
            inputs: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'input', objectType: 'InputItem', prisma, userData }),
            outputs: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'output', objectType: 'OutputItem', prisma, userData }),
            nodes: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'node', objectType: 'Node', prisma, userData }),
            nodeLinks: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'nodeLink', objectType: 'NodeLink', prisma, userData }),
            translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
        }
    }
}

/**
 * Handles authorized creates, updates, and deletes
 */
const mutater = (): Mutater<
    Routine,
    { graphql: RoutineCreateInput, db: Prisma.routineUpsertArgs['create'] },
    { graphql: RoutineUpdateInput, db: Prisma.routineUpsertArgs['update'] },
    { graphql: RoutineCreateInput, db: Prisma.routineCreateWithoutTransfersInput },
    { graphql: RoutineUpdateInput, db: Prisma.routineUpdateWithoutTransfersInput }
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            const [simplicity, complexity] = await calculateComplexity(prisma, userData.languages, data);
            const base = await shapeBase(prisma, userData, data, true);
            return {
                id: data.id,
                ...base.root,
                hasCompleteVersion: data.isComplete ? true : false,
                versions: {
                    create: {
                        ...base.version,
                        complexity,
                        simplicity,
                    }
                }
            }
        },
        update: async ({ data, prisma, userData, where }) => {
            const [simplicity, complexity] = await calculateComplexity(prisma, userData.languages, data, data.versionId);
            const base = await shapeBase(prisma, userData, data, false);
            // Determine hasCompleteVersion. 
            let hasCompleteVersion: boolean | undefined = undefined;
            // If setting isComplete to true, set hasCompleteVersion to true
            if (data.isComplete === true) hasCompleteVersion = true;
            // Otherwise, query for existing versions
            else {
                const existingVersions = await prisma.routine_version.findMany({
                    where: { rootId: where.id },
                    select: { id: true, isComplete: true }
                });
                // Set hasCompleteVersion to true if any version is complete. 
                // Exclude the version being updated, if it exists
                if (existingVersions.some(v => v.id !== data.versionId && v.isComplete)) hasCompleteVersion = true;
                // If none of the existing versions are complete, set hasCompleteVersion to false
                else hasCompleteVersion = false;
            }
            return {
                ...base.root,
                //
                hasCompleteVersion,
                // parent: data.parentId ? { connect: { id: data.parentId } } : undefined,
                creator: { connect: { id: userData.id } },
                organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
                // If versionId is provided, update that version. 
                // Otherwise, versionLabel is provided, so create new version with that label
                versions: {
                    ...(data.versionId ? {
                        update: {
                            where: { id: data.versionId },
                            data: {
                                ...base.version,
                                complexity: complexity,
                                simplicity: simplicity,
                            }
                        }
                    } : {
                        create: {
                            ...base.version,
                            versionLabel: data.versionLabel as string,
                            complexity: complexity,
                            simplicity: simplicity,
                        }
                    })
                },
            }
        },
        relCreate: (...args) => mutater().shape.create(...args),
        relUpdate: (...args) => mutater().shape.update(...args),
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
            //     const hadCompletedVersion = validator().hasCompletedVersion(permissionsData as any);
            //     const isPublic = input.isPrivate !== undefined ? !input.isPrivate : wasPublic;
            //     const hasCompletedVersion = asdfasdfasdf
            //     // Check if new version was created
            //     if (input.versionLabel) {
            //         Trigger(prisma, userData.languages).objectNewVersion(
            //             userData.id,
            //             'Routine',
            //             u.id,
            //             owner,
            //             hasOriginalOwner,
            //             hadCompletedVersion && wasPublic,
            //             hasCompletedVersion && isPublic
            //         );
            //     }
            // }
        },
    },
    yup: { create: routinesCreate, update: routinesUpdate },
})

const displayer = (): Displayer<
    Prisma.routine_versionSelect,
    Prisma.routine_versionGetPayload<{ select: { [K in keyof Required<Prisma.routine_versionSelect>]: true } }>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, title: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'title', languages),
})

export const RoutineVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.routine_version,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'RoutineVersion' as GraphQLModelType,
    validate: validator(),
    calculateComplexity,
    validateNodePositions,
})