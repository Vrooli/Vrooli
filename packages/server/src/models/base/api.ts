import { ApiSortBy, apiValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils/defaultPermissions";
import { oneIsPublic } from "../../utils/oneIsPublic";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { labelShapeHelper, ownerFields, preShapeRoot, PreShapeRootResult, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers/afterMutationsRoot";
import { getSingleTypePermissions } from "../../validators/permissions";
import { ApiFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ApiModelInfo, ApiModelLogic, ApiVersionModelLogic, BookmarkModelLogic, ReactionModelLogic, TeamModelLogic, UserModelLogic, ViewModelLogic } from "./types";

type ApiPre = PreShapeRootResult;

const __typename = "Api" as const;
export const ApiModel: ApiModelLogic = ({
    __typename,
    dbTable: "api",
    display: () => rootObjectDisplay(ModelMap.get<ApiVersionModelLogic>("ApiVersion")),
    format: ApiFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<ApiPre> => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ApiPre;
                return {
                    id: data.id,
                    isPrivate: data.isPrivate,
                    permissions: noNull(data.permissions) ?? JSON.stringify({}),
                    createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "apis", isCreate: true, objectType: __typename, data, ...rest })),
                    parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "ApiVersion", parentRelationshipName: "forks", data, ...rest }),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "ApiVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Api", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Api", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ApiPre;
                return {
                    isPrivate: noNull(data.isPrivate),
                    permissions: noNull(data.permissions),
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "apis", isCreate: false, objectType: __typename, data, ...rest })),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ApiVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Api", relation: "tags", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Api", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
            },
        },
        yup: apiValidation,
    },
    search: {
        defaultSort: ApiSortBy.ScoreDesc,
        sortBy: ApiSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            hasCompleteVersion: true,
            issuesId: true,
            labelsIds: true,
            maxScore: true,
            maxBookmarks: true,
            maxViews: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            ownedByTeamId: true,
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
                { versions: { some: "transNameWrapped" } },
                { versions: { some: "transSummaryWrapped" } },
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
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted === false,
        isPublic: (data, ...rest) =>
            data.isPrivate === false &&
            data.isDeleted === false &&
            (
                (data.ownedByUser === null && data.ownedByTeam === null) ||
                oneIsPublic<ApiModelInfo["PrismaSelect"]>([
                    ["ownedByTeam", "Team"],
                    ["ownedByUser", "User"],
                ], data, ...rest)
            ),
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data?.ownedByTeam,
            User: data?.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isPrivate: true,
            permissions: true,
            createdBy: "User",
            ownedByTeam: "Team",
            ownedByUser: "User",
            versions: ["ApiVersion", ["root"]],
        }),
        visibility: {
            //TODO for morning: all visiblity private/public need to be updated. Some are just blank when they should at MINIMUM look like the ones below, and ones like this one should also be checking that either the owners are both null, or the owner is also public/private
            //TODO 2: Finish updating StandardVersionSelectSwitch to be generic. Use it for connecting code to single-step routine. Make sure it still works with inputs, and that we can set "isInternal" through the create standard popup. Also improve look of RoutineUpsert for single-step routines, so that we use a dropdown to select the type (e.g. api, data converter (code), smart contract, instruction, LLM generation, instruction (i.e. does nothing but show description and details), etc.)
            private: {
                isDeleted: false,
                OR: [
                    { isPrivate: true },
                    { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").validate().visibility.private },
                    { ownedByUser: ModelMap.get<UserModelLogic>("User").validate().visibility.private },
                ],
            },
            public: {
                isDeleted: false,
                isPrivate: false,
                OR: [
                    { ownedByTeam: null, ownedByUser: null },
                    { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").validate().visibility.public },
                    { ownedByUser: ModelMap.get<UserModelLogic>("User").validate().visibility.public },
                ],
            },
            owner: (userId) => ({
                OR: [
                    { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                    { ownedByUser: { id: userId } },
                ],
            }),
        },
    }),
});
