import { routineValidation } from "@shared/validation";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Trigger } from "../events";
import { Routine, RoutineSearchInput, RoutineCreateInput, RoutineUpdateInput, RoutineSortBy, SessionUser, RoutineYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { getSingleTypePermissions } from "../validators";
import { SelectWrap } from "../builders/types";
import { padSelect, permissionsSelectHelper } from "../builders";
import { oneIsPublic } from "../utils";
import { RoutineVersionModel } from "./routineVersion";
import { getLabels } from "../getters";

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

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: RoutineCreateInput | RoutineUpdateInput, isAdd: boolean) => {
    return {
        // root: {
        //     isPrivate: data.isPrivate ?? undefined,
        //     hasCompleteVersion: (data.isComplete === true) ? true : (data.isComplete === false) ? false : undefined,
        //     completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
        //     project: data.projectId ? { connect: { id: data.projectId } } : undefined,
        //     tags: await tagRelationshipBuilder(prisma, userData, data, 'Routine', isAdd),
        //     permissions: JSON.stringify({}),
        // },
        // version: {
        //     isAutomatable: data.isAutomatable ?? undefined,
        //     isComplete: data.isComplete ?? undefined,
        //     isInternal: data.isInternal ?? undefined,
        //     versionLabel: data.versionLabel ?? undefined,
        //     resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
        //     inputs: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'input', objectType: 'RoutineVersionInput', prisma, userData }),
        //     outputs: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'output', objectType: 'RoutineVersionOutput', prisma, userData }),
        //     nodes: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'node', objectType: 'Node', prisma, userData }),
        //     nodeLinks: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'nodeLink', objectType: 'NodeLink', prisma, userData }),
        //     translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
        // }
    }
}

const __typename = 'Routine' as const;
type Permissions = Pick<RoutineYou, 'canComment' | 'canDelete' | 'canEdit' | 'canStar' | 'canView' | 'canVote'>;
const suppFields = ['you.canComment', 'you.canDelete', 'you.canEdit', 'you.canStar', 'you.canView', 'you.canVote', 'you.isStarred', 'you.isUpvoted', 'you.isViewed', 'translatedName'] as const;
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
    display: {
        select: () => ({
            id: true,
            versions: {
                where: { isPrivate: false },
                orderBy: { versionIndex: 'desc' },
                take: 1,
                select: RoutineVersionModel.display.select(),
            }
        }),
        label: (select, languages) => select.versions.length > 0 ?
            RoutineVersionModel.display.label(select.versions[0] as any, languages) : '',
    },
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
            starredBy: 'User',
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
            starredBy: 'User',
            stats: 'StatsRoutine',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'RoutineVersion',
            viewedBy: 'View',
        },
        joinMap: { labels: 'label', tags: 'tag', starredBy: 'user' },
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
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                    'you.isViewed': await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    'you.isUpvoted': await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    'translatedName': await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'project.translatedName')
                }
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => {
                // const [simplicity, complexity] = await calculateComplexity(prisma, userData.languages, data);
                // const base = await shapeBase(prisma, userData, data, true);
                // return {
                //     id: data.id,
                //     ...base.root,
                //     hasCompleteVersion: data.isComplete ? true : false,
                //     versions: {
                //         create: {
                //             ...base.version,
                //             complexity,
                //             simplicity,
                //         }
                //     }
                // }
                return {} as any;
            },
            update: async ({ data, prisma, userData, where }) => {
                // const [simplicity, complexity] = await calculateComplexity(prisma, userData.languages, data, data.versionId);
                // const base = await shapeBase(prisma, userData, data, false);
                // // Determine hasCompleteVersion. 
                // let hasCompleteVersion: boolean | undefined = undefined;
                // // If setting isComplete to true, set hasCompleteVersion to true
                // if (data.isComplete === true) hasCompleteVersion = true;
                // // Otherwise, query for existing versions
                // else {
                //     const existingVersions = await prisma.routine_version.findMany({
                //         where: { rootId: where.id },
                //         select: { id: true, isComplete: true }
                //     });
                //     // Set hasCompleteVersion to true if any version is complete. 
                //     // Exclude the version being updated, if it exists
                //     if (existingVersions.some(v => v.id !== data.versionId && v.isComplete)) hasCompleteVersion = true;
                //     // If none of the existing versions are complete, set hasCompleteVersion to false
                //     else hasCompleteVersion = false;
                // }
                // return {
                //     ...base.root,
                //     //
                //     hasCompleteVersion,
                //     // parent: data.parentId ? { connect: { id: data.parentId } } : undefined,
                //     creator: { connect: { id: userData.id } },
                //     organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                //     user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
                //     // If versionId is provided, update that version. 
                //     // Otherwise, versionLabel is provided, so create new version with that label
                //     versions: {
                //         ...(data.versionId ? {
                //             update: {
                //                 where: { id: data.versionId },
                //                 data: {
                //                     ...base.version,
                //                     complexity: complexity,
                //                     simplicity: simplicity,
                //                 }
                //             }
                //         } : {
                //             create: {
                //                 ...base.version,
                //                 versionLabel: data.versionLabel as string,
                //                 complexity: complexity,
                //                 simplicity: simplicity,
                //             }
                //         })
                //     },
                // }
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
            labelsId: true,
            maxScore: true,
            maxStars: true,
            maxViews: true,
            minScore: true,
            minStars: true,
            minViews: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
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
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            data.isInternal === false &&
            oneIsPublic<Prisma.routineSelect>(data, [
                ['ownedByOrganization', 'Organization'],
                ['ownedByUser', 'User'],
            ], languages),
        isTransferable: true,
        hasCompletedVersion: (data) => data.hasCompleteVersion === true,
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
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            canComment: () => !isDeleted && (isAdmin || isPublic),
            canDelete: () => isAdmin && !isDeleted,
            canEdit: () => isAdmin && !isDeleted,
            canReport: () => !isAdmin && !isDeleted && isPublic,
            canRun: () => !isDeleted && (isAdmin || isPublic),
            canStar: () => !isDeleted && (isAdmin || isPublic),
            canView: () => !isDeleted && (isAdmin || isPublic),
            canVote: () => !isDeleted && (isAdmin || isPublic),
        }),
        permissionsSelect: (...params) => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isPrivate: true,
            isInternal: true,
            permissions: true,
            createdBy: padSelect({ id: true }),
            ...permissionsSelectHelper({
                ownedByOrganization: 'Organization',
                ownedByUser: 'User',
            }, ...params),
        }),
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
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
    },
})