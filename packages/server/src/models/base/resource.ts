import { MaxObjects, ResourceSortBy, resourceValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { ResourceFormat } from "../formats";
import { ModelLogic } from "../types";
import { ResourceListModel } from "./resourceList";
import { ResourceListModelLogic, ResourceModelLogic } from "./types";

const __typename = "Resource" as const;
const suppFields = [] as const;
export const ResourceModel: ModelLogic<ResourceModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.resource,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
    },
    format: ResourceFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                link: data.link,
                usedFor: data.usedFor,
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "ResourceList", parentRelationshipName: "resources", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                link: noNull(data.link),
                usedFor: noNull(data.usedFor),
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "resources", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: resourceValidation,
    },
    search: {
        defaultSort: ResourceSortBy.IndexAsc,
        sortBy: ResourceSortBy,
        searchFields: {
            createdTimeFrame: true,
            resourceListId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                "linkWrapped",
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            list: "ResourceList",
        }),
        permissionResolvers: (params) => ResourceListModel.validate.permissionResolvers({ ...params, data: params.data.list as any }),
        owner: (data, userId) => ResourceListModel.validate.owner(data?.list as ResourceListModelLogic["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceModelLogic["PrismaSelect"]>([["list", "ResourceList"]], ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ list: ResourceListModel.validate.visibility.owner(userId) }),
        },
    },
});
