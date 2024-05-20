import { MaxObjects, SmartContractSortBy, smartContractValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { getLabels } from "../../getters";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { PreShapeRootResult, labelShapeHelper, ownerFields, preShapeRoot, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { SmartContractFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, OrganizationModelLogic, ReactionModelLogic, SmartContractModelInfo, SmartContractModelLogic, SmartContractVersionModelLogic, ViewModelLogic } from "./types";

type SmartContractPre = PreShapeRootResult;

const __typename = "SmartContract" as const;
export const SmartContractModel: SmartContractModelLogic = ({
    __typename,
    dbTable: "smart_contract",
    display: () => rootObjectDisplay(ModelMap.get<SmartContractVersionModelLogic>("SmartContractVersion")),
    format: SmartContractFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<SmartContractPre> => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as SmartContractPre;
                return {
                    id: data.id,
                    isPrivate: data.isPrivate,
                    permissions: noNull(data.permissions) ?? JSON.stringify({}),
                    createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "smartContracts", isCreate: true, objectType: __typename, data, ...rest })),
                    parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "SmartContractVersion", parentRelationshipName: "forks", data, ...rest }),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "SmartContractVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "SmartContract", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "SmartContract", data, ...rest }),
                }
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as SmartContractPre;
                return {
                    isPrivate: noNull(data.isPrivate),
                    permissions: noNull(data.permissions),
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "smartContracts", isCreate: false, objectType: __typename, data, ...rest })),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "SmartContractVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "SmartContract", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "SmartContract", data, ...rest }),
                }
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
            },
        },
        yup: smartContractValidation,
    },
    search: {
        defaultSort: SmartContractSortBy.ScoreDesc,
        sortBy: SmartContractSortBy,
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
                { versions: { some: "transNameWrapped" } },
                { versions: { some: "transDescriptionWrapped" } },
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
                    "translatedName": await getLabels(ids, __typename, userData?.languages ?? ["en"], "smartContract.translatedName"),
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
            oneIsPublic<SmartContractModelInfo["PrismaSelect"]>([
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
            versions: ["SmartContractVersion", ["root"]],
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
