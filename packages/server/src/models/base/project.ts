import { MaxObjects, ProjectSortBy, projectValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { getLabels } from "../../getters";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { labelShapeHelper, ownerFields, preShapeRoot, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers";
import { getSingleTypePermissions, handlesCheck } from "../../validators";
import { ProjectFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, OrganizationModelLogic, ProjectModelInfo, ProjectModelLogic, ProjectVersionModelLogic, ReactionModelLogic, ViewModelLogic } from "./types";

const __typename = "Project" as const;
export const ProjectModel: ProjectModelLogic = ({
    __typename,
    dbTable: "project",
    display: () => rootObjectDisplay(ModelMap.get<ProjectVersionModelLogic>("ProjectVersion")),
    format: ProjectFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update, Delete, userData }) => {
                await handlesCheck(__typename, Create, Update, userData.languages);
                const maps = await preShapeRoot({ Create, Update, Delete, userData, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                handle: noNull(data.handle),
                isPrivate: data.isPrivate,
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "projects", isCreate: true, objectType: __typename, data, ...rest })),
                parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersion", parentRelationshipName: "forks", data, ...rest }),
                versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "ProjectVersion", parentRelationshipName: "root", data, ...rest }),
                tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Project", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Project", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                handle: noNull(data.handle),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "projects", isCreate: false, objectType: __typename, data, ...rest })),
                versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ProjectVersion", parentRelationshipName: "root", data, ...rest }),
                tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Project", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Project", data, ...rest }),
            }),
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
                    translatedName: await getLabels(ids, __typename, userData?.languages ?? ["en"], "project.translatedName"),
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
            oneIsPublic<ProjectModelInfo["PrismaSelect"]>([
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
            isPrivate: true,
            permissions: true,
            createdBy: "User",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            versions: ["ProjectVersion", ["root"]],
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
