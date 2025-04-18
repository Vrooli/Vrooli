import { DEFAULT_LANGUAGE, MaxObjects, ProjectSortBy, projectValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { getLabels } from "../../getters/getLabels.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay.js";
import { labelShapeHelper } from "../../utils/shapes/labelShapeHelper.js";
import { ownerFields } from "../../utils/shapes/ownerFields.js";
import { preShapeRoot, type PreShapeRootResult } from "../../utils/shapes/preShapeRoot.js";
import { tagShapeHelper } from "../../utils/shapes/tagShapeHelper.js";
import { afterMutationsRoot } from "../../utils/triggers/afterMutationsRoot.js";
import { handlesCheck } from "../../validators/handlesCheck.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { ProjectFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, ProjectModelInfo, ProjectModelLogic, ProjectVersionModelLogic, ReactionModelLogic, TeamModelLogic, ViewModelLogic } from "./types.js";

type ProjectPre = PreShapeRootResult;

const __typename = "Project" as const;
export const ProjectModel: ProjectModelLogic = ({
    __typename,
    dbTable: "project",
    display: () => rootObjectDisplay(ModelMap.get<ProjectVersionModelLogic>("ProjectVersion")),
    format: ProjectFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update, Delete, userData }): Promise<ProjectPre> => {
                await handlesCheck(__typename, Create, Update);
                const maps = await preShapeRoot({ Create, Update, Delete, userData, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ProjectPre;
                return {
                    id: data.id,
                    handle: noNull(data.handle),
                    isPrivate: data.isPrivate,
                    permissions: noNull(data.permissions) ?? JSON.stringify({}),
                    createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "projects", isCreate: true, objectType: __typename, data, ...rest })),
                    parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersion", parentRelationshipName: "forks", data, ...rest }),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "ProjectVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Project", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Project", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ProjectPre;
                return {
                    handle: noNull(data.handle),
                    isPrivate: noNull(data.isPrivate),
                    permissions: noNull(data.permissions),
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "projects", isCreate: false, objectType: __typename, data, ...rest })),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ProjectVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Project", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Project", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
            },
        },
        yup: projectValidation,
    },
    search: {
        defaultSort: ProjectSortBy.ScoreDesc,
        sortBy: ProjectSortBy,
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
                { versions: { some: "transDescriptionWrapped" } },
                { versions: { some: "transNameWrapped" } },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ProjectModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, userData?.languages ?? [DEFAULT_LANGUAGE], "project.translatedName"),
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, ...rest) =>
            data.isPrivate === false &&
            data.isDeleted === false &&
            (
                (data.ownedByUser === null && data.ownedByTeam === null) ||
                oneIsPublic<ProjectModelInfo["DbSelect"]>([
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
            versions: ["ProjectVersion", ["root"]],
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId) },
                        { ownedByUser: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Owned objects
                        {
                            OR: (useVisibility("Project", "Own", data) as { OR: object[] }).OR,
                        },
                        // Public objects
                        {
                            isPrivate: false, // Can't be private
                            OR: (useVisibility("Project", "Public", data) as { OR: object[] }).OR,
                        },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: true,  // Must be private
                    OR: (useVisibility("Project", "Own", data) as { OR: object[] }).OR,
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false,  // Must be public
                    OR: (useVisibility("Project", "Own", data) as { OR: object[] }).OR,
                };
            },
            public: function getPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false, // Can't be private
                    OR: [
                        { ownedByTeam: null, ownedByUser: null },
                        { ownedByTeam: useVisibility("Team", "Public", data) },
                        { ownedByUser: { isPrivate: false, isPrivateProjects: false } },
                    ],
                };
            },
        },
    }),
});
