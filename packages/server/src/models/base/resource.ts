import { MaxObjects, ResourceSortBy, getTranslation, resourceValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { ResourceFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { ResourceListModelInfo, ResourceListModelLogic, ResourceModelInfo, ResourceModelLogic } from "./types.js";

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
        owner: (data, userId) => ModelMap.get<ResourceListModelLogic>("ResourceList").validate().owner(data?.list as ResourceListModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceModelInfo["DbSelect"]>([["list", "ResourceList"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    list: useVisibility("ResourceList", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    list: useVisibility("ResourceList", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    list: useVisibility("ResourceList", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    list: useVisibility("ResourceList", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    list: useVisibility("ResourceList", "Public", data),
                };
            },
        },
    }),
});
