import { MaxObjects, ResourceSortBy, getTranslation, resourceValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { ResourceFormat } from "../formats";
import { ResourceListModelInfo, ResourceListModelLogic, ResourceModelInfo, ResourceModelLogic } from "./types";

const __typename = "Resource" as const;
export const ResourceModel: ResourceModelLogic = ({
    __typename,
    dbTable: "resource",
    dbTranslationTable: "resource_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
    }),
    format: ResourceFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                link: data.link,
                usedFor: data.usedFor,
                list: await shapeHelper({ relation: "list", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "resources", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                link: noNull(data.link),
                usedFor: noNull(data.usedFor),
                list: await shapeHelper({ relation: "list", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "resources", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
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
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            list: "ResourceList",
        }),
        permissionResolvers: (params) => ModelMap.get<ResourceListModelLogic>("ResourceList").validate().permissionResolvers({ ...params, data: params.data.list as any }),
        owner: (data, userId) => ModelMap.get<ResourceListModelLogic>("ResourceList").validate().owner(data?.list as ResourceListModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceModelInfo["PrismaSelect"]>([["list", "ResourceList"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({ list: ModelMap.get<ResourceListModelLogic>("ResourceList").validate().visibility.owner(userId) }),
        },
    }),
});
