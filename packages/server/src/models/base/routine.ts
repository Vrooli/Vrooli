import { MaxObjects, RoutineSortBy, routineValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { getLabels } from "../../getters";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { PreShapeRootResult, labelShapeHelper, ownerFields, preShapeRoot, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { RoutineFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, OrganizationModelLogic, ReactionModelLogic, RoutineModelInfo, RoutineModelLogic, RoutineVersionModelLogic, ViewModelLogic } from "./types";

type RoutinePre = PreShapeRootResult;

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

const __typename = "Routine" as const;
export const RoutineModel: RoutineModelLogic = ({
    __typename,
    dbTable: "routine",
    display: () => rootObjectDisplay(ModelMap.get<RoutineVersionModelLogic>("RoutineVersion")),
    format: RoutineFormat,
    mutate: {
        shape: {
            // TODO for morning 2: need to create helper to handle version pre/post logic. These should 
            // also support calling Trigger, and also version index logic. I started implementing this (the version index logic) somewhere before, 
            // maybe models/routineVersion.
            pre: async (params): Promise<RoutinePre> => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as RoutinePre;
                return {
                    id: data.id,
                    isInternal: noNull(data.isInternal),
                    isPrivate: data.isPrivate,
                    permissions: noNull(data.permissions) ?? JSON.stringify({}),
                    createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "routines", isCreate: true, objectType: __typename, data, ...rest })),
                    parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "forks", data, ...rest }),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "RoutineVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Routine", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Routine", data, ...rest }),
                }
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as RoutinePre;
                return {
                    isInternal: noNull(data.isInternal),
                    isPrivate: noNull(data.isPrivate),
                    permissions: noNull(data.permissions),
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "routines", isCreate: false, objectType: __typename, data, ...rest })),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RoutineVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Routine", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Routine", data, ...rest }),
                }
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
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
        },
        searchStringQuery: () => ({
            OR: [
                "tagsWrapped",
                "labelsWrapped",
                { versions: { some: "transDescriptionWrapped" } },
                { versions: { some: "transNameWrapped" } },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                    "translatedName": await getLabels(ids, __typename, userData?.languages ?? ["en"], "project.translatedName"),
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            data.isInternal === false &&
            oneIsPublic<RoutineModelInfo["PrismaSelect"]>([
                ["ownedByOrganization", "Organization"],
                ["ownedByUser", "User"],
            ], data, ...rest),
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data?.ownedByOrganization,
            User: data?.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isInternal: true,
            isPrivate: true,
            permissions: true,
            createdBy: "User",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            versions: ["RoutineVersion", ["root"]],
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId) },
                ],
            }),
        },
    }),
});
