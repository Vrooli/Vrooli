import { routineValidation } from "@shared/validation";
import { BookmarkModel } from "./bookmark";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Trigger } from "../events";
import { Routine, RoutineSearchInput, RoutineCreateInput, RoutineUpdateInput, RoutineSortBy, SessionUser, RoutineYou, PrependString, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { getSingleTypePermissions } from "../validators";
import { SelectWrap } from "../builders/types";
import { defaultPermissions, labelShapeHelper, oneIsPublic, preHasPublics, tagShapeHelper } from "../utils";
import { RoutineVersionModel } from "./routineVersion";
import { getLabels, getLogic } from "../getters";
import { rootObjectDisplay } from "../utils/rootObjectDisplay";
import { noNull, shapeHelper } from "../builders";

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
//                 nodeType: true,
//                 end: {
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
//                                         name: true,
//                                         language: true,
//                                     }
//                                 }
//                             }
//                         }
//                     }
//                 },
//                 routineList: {
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
//                                         name: true,
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
//                         name: true,
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
//                                 name: true,
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
//                                 name: true,
//                                 language: true,
//                             }
//                         }
//                     }
//                 },
//                 translations: {
//                     select: {
//                         description: true,
//                         name: true,
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
//                 name: true,
//                 language: true,
//             }
//         }
//     },
//     // validateSelect: {
//     //     nodes: {
//     //         select: {
//     //             routineList: {
//     //                 select: {
//     //                     items: {
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

const __typename = 'Routine' as const;
type Permissions = Pick<RoutineYou, 'canComment' | 'canDelete' | 'canUpdate' | 'canBookmark' | 'canRead' | 'canVote'>;
const suppFields = ['you', 'translatedName'] as const;
export const RoutineModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RoutineCreateInput,
    GqlUpdate: RoutineUpdateInput,
    GqlModel: Routine,
    GqlSearch: RoutineSearchInput,
    GqlSort: RoutineSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.routineUpsertArgs['create'],
    PrismaUpdate: Prisma.routineUpsertArgs['update'],
    PrismaModel: Prisma.routineGetPayload<SelectWrap<Prisma.routineSelect>>,
    PrismaSelect: Prisma.routineSelect,
    PrismaWhere: Prisma.routineWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine,
    display: rootObjectDisplay(RoutineVersionModel),
    format: {
        gqlRelMap: {
            __typename,
            createdBy: 'User',
            owner: {
                ownedByUser: 'User',
                ownedByOrganization: 'Organization',
            },
            forks: 'Routine',
            issues: 'Issue',
            labels: 'Label',
            parent: 'Routine',
            bookmarkedBy: 'User',
            tags: 'Tag',
            versions: 'RoutineVersion',
        },
        prismaRelMap: {
            __typename,
            createdBy: 'User',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            parent: 'RoutineVersion',
            questions: 'Question',
            quizzes: 'Quiz',
            issues: 'Issue',
            labels: 'Label',
            pullRequests: 'PullRequest',
            bookmarkedBy: 'User',
            stats: 'StatsRoutine',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'RoutineVersion',
            viewedBy: 'View',
        },
        joinMap: { labels: 'label', tags: 'tag', bookmarkedBy: 'user' },
        countFields: {
            forksCount: true,
            issuesCount: true,
            pullRequestsCount: true,
            questionsCount: true,
            quizzesCount: true,
            transfersCount: true,
            versionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    },
                    'translatedName': await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'project.translatedName')
                }
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList, prisma, userData }) => {
                const maps = await preHasPublics({ createList, updateList, objectType: __typename, prisma, userData });
                return { ...maps }
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isInternal: noNull(data.isInternal),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].hasCompleteVersion[data.id],
                ...(await shapeHelper({ relation: 'ownedByUser', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'User', parentRelationshipName: 'routinesCreated', data, ...rest })),
                ...(await shapeHelper({ relation: 'ownedByOrganization', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName: 'routines', data, ...rest })),
                ...(await shapeHelper({ relation: 'parent', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'RoutineVersion', parentRelationshipName: 'forks', data, ...rest })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersion', parentRelationshipName: 'root', data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Routine', relation: 'tags', data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Routine', relation: 'labels', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isInternal: noNull(data.isInternal),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].hasCompleteVersion[data.id],
                ...(await shapeHelper({ relation: 'ownedByUser', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'User', parentRelationshipName: 'routinesCreated', data, ...rest })),
                ...(await shapeHelper({ relation: 'ownedByOrganization', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName: 'routines', data, ...rest })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'RoutineVersion', parentRelationshipName: 'root', data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Routine', relation: 'tags', data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Routine', relation: 'labels', data, ...rest })),
            }),
        },
        trigger: {
            onCreated: ({ created, prisma, userData }) => {
                for (const c of created) {
                    // Trigger(prisma, userData.languages).createRoutine(userData.id, c.id as string);
                }
            },
            onUpdated: ({ authData, prisma, updated, updateInput, userData }) => {
                // // Initialize transfers, if any
                // asdfasdfasfd
                // Handle objectUpdated trigger
                // Loop through updated items
                for (let i = 0; i < updated.length; i++) {
                    // Get data for current item
                    const u = updated[i];
                    const input = updateInput[i];
                    const permissionsData = authData[u.id];
                    // Use permissions to find orignial data
                    const { Organization, User } = RoutineModel.validate!.owner(permissionsData as any);
                    const originalOwner: { __typename: 'Organization' | 'User', id: string } | null = Organization ?
                        { __typename: 'Organization', id: Organization.id } :
                        User ? { __typename: 'User', id: User.id } : null;
                    const wasCompleteAndPublic =
                        RoutineModel.validate!.isPublic(permissionsData as any, userData.languages) &&
                        RoutineModel.validate!.hasCompleteVersion(permissionsData as any);
                    const hasParent = asdfasdf
                    const originalProjectId = asdfasdf;
                    // Use input to determine which data has changed
                    const hasCompleteAndPublic = asdfasdf
                    const projectId = asdfasdf
                    // Trigger objectUpdated
                    Trigger(prisma, userData.languages).objectUpdated(
                        updatedById: userData.id,
                        hasCompleteAndPublic,
                        hasParent,
                        owner,
                        objectId: u.id,
                        objectType: __typename,
                        originalProjectId,
                        projectId,
                        wasCompleteAndPublic,
                    );
                }
            },
        },
        yup: routineValidation,
    },
    search: {
        defaultSort: RoutineSortBy.ScoreDesc,
        sortBy: RoutineSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            hasCompleteVersion: true,
            isInternal: true,
            issuesId: true,
            labelsIds: true,
            maxScore: true,
            maxBookmarks: true,
            maxViews: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
            pullRequestsId: true,
            tags: true,
            translationLanguagesLatestVersion: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'tagsWrapped',
                'labelsWrapped',
                { versions: { some: 'transDescriptionWrapped' } },
                { versions: { some: 'transNameWrapped' } }
            ]
        })
    },
    validate: {
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            data.isInternal === false &&
            oneIsPublic<Prisma.routineSelect>(data, [
                ['ownedByOrganization', 'Organization'],
                ['ownedByUser', 'User'],
            ], languages),
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isInternal: true,
            isPrivate: true,
            permissions: true,
            createdBy: 'User',
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
            versions: ['RoutineVersion', ['root']],
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})