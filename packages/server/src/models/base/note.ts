import { MaxObjects, NoteSortBy, noteValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { labelShapeHelper, ownerFields, preShapeRoot, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { NoteFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, NoteModelLogic, NoteVersionModelLogic, OrganizationModelLogic, ReactionModelLogic, ViewModelLogic } from "./types";

const __typename = "Note" as const;
export const NoteModel: NoteModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.note,
    display: () => rootObjectDisplay(ModelMap.get<NoteVersionModelLogic>("NoteVersion")),
    format: NoteFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: data.isPrivate,
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "notes", isCreate: true, objectType: __typename, data, ...rest })),
                parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "NoteVersion", parentRelationshipName: "forks", data, ...rest }),
                versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "NoteVersion", parentRelationshipName: "root", data, ...rest }),
                tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Note", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Note", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "notes", isCreate: false, objectType: __typename, data, ...rest })),
                versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "NoteVersion", parentRelationshipName: "root", data, ...rest }),
                tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Note", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Note", data, ...rest }),
            }),
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
            },
        },
        yup: noteValidation,
    },
    search: {
        defaultSort: NoteSortBy.DateUpdatedDesc,
        sortBy: NoteSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            maxBookmarks: true,
            maxScore: true,
            minBookmarks: true,
            minScore: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
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
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: () => true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted === true,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data?.ownedByOrganization,
            User: data?.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            permissions: true,
            createdBy: "User",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            versions: ["NoteVersion", ["root"]],
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
