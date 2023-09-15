import { MaxObjects, ProjectSortBy, projectValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { getLabels } from "../../getters";
import { defaultPermissions, labelShapeHelper, onCommonRoot, oneIsPublic, ownerShapeHelper, preShapeRoot, tagShapeHelper } from "../../utils";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { getSingleTypePermissions, handlesCheck } from "../../validators";
import { ProjectFormat } from "../format/project";
import { ModelLogic } from "../types";
import { BookmarkModel } from "./bookmark";
import { OrganizationModel } from "./organization";
import { ProjectVersionModel } from "./projectVersion";
import { ReactionModel } from "./reaction";
import { ProjectModelLogic } from "./types";
import { ViewModel } from "./view";

const __typename = "Project" as const;
const suppFields = ["you", "translatedName"] as const;
export const ProjectModel: ModelLogic<ProjectModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.project,
    display: rootObjectDisplay(ProjectVersionModel),
    format: ProjectFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update, Delete, prisma, userData }) => {
                await handlesCheck(prisma, __typename, Create, Update, userData.languages);
                const maps = await preShapeRoot({ Create, Update, Delete, prisma, userData, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                handle: noNull(data.handle),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "projects", isCreate: true, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "ProjectVersion", parentRelationshipName: "forks", data, ...rest })),
                ...(await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ProjectVersion", parentRelationshipName: "root", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Project", relation: "tags", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Project", relation: "labels", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                handle: noNull(data.handle),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "projects", isCreate: false, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "ProjectVersion", parentRelationshipName: "root", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Project", relation: "tags", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Project", relation: "labels", data, ...rest })),
            }),
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonRoot({ ...params, objectType: __typename });
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
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, prisma, userData?.languages ?? ["en"], "project.translatedName"),
                };
            },
        },
    },
    validate: {
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<Prisma.projectSelect>(data, [
                ["ownedByOrganization", "Organization"],
                ["ownedByUser", "User"],
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
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
