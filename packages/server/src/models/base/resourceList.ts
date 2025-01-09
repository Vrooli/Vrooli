import { MaxObjects, ModelType, ResourceListFor, ResourceListSearchInput, ResourceListSortBy, getTranslation, resourceListValidation, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility, useVisibilityMapper } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { ResourceListFormat } from "../formats";
import { ResourceListModelInfo, ResourceListModelLogic } from "./types";

const forMapper: { [key in ResourceListFor]: keyof Prisma.resource_listUpsertArgs["create"] } = {
    ApiVersion: "apiVersion",
    CodeVersion: "codeVersion",
    FocusMode: "focusMode",
    Post: "post",
    ProjectVersion: "projectVersion",
    RoutineVersion: "routineVersion",
    StandardVersion: "standardVersion",
    Team: "team",
};
const reversedForMapper = Object.fromEntries(
    Object.entries(forMapper).map(([key, value]) => [value, key]),
);

const __typename = "ResourceList" as const;
export const ResourceListModel: ResourceListModelLogic = ({
    __typename,
    dbTable: "resource_list",
    dbTranslationTable: "resource_list_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
    }),
    format: ResourceListFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                [forMapper[data.listForType]]: { connect: { id: data.listForConnect } },
                resources: await shapeHelper({ relation: "resources", relTypes: ["Create"], isOneToOne: false, objectType: "Resource", parentRelationshipName: "list", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                resources: await shapeHelper({ relation: "resources", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "Resource", parentRelationshipName: "list", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: resourceListValidation,
    },
    search: {
        defaultSort: ResourceListSortBy.IndexAsc,
        sortBy: ResourceListSortBy,
        searchFields: {
            apiVersionId: true,
            codeVersionId: true,
            createdTimeFrame: true,
            postId: true,
            projectVersionId: true,
            routineVersionId: true,
            standardVersionId: true,
            teamId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            focusModeId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
            ],
        }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value, key as ModelType])),
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => {
            if (!data) return {};
            const [resourceOnType, resourceOnData] = findFirstRel(data, Object.values(forMapper));
            if (!resourceOnType || !resourceOnData) return {};
            return ModelMap.get(uppercaseFirstLetter(resourceOnType) as ModelType).validate().owner(resourceOnData, userId);
        },
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceListModelInfo["DbSelect"]>(Object.entries(forMapper).map(([key, value]) => [value, key as ModelType]), ...rest),
        visibility: {
            own: function getOwn(data) {
                const searchInput = data.searchInput as ResourceListSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "Own", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    OR: [
                        ...useVisibilityMapper("Own", data, forMapper, false),
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("ResourceList", "Own", data),
                        useVisibility("ResourceList", "Public", data),
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                const searchInput = data.searchInput as ResourceListSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "OwnPrivate", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    OR: [
                        ...useVisibilityMapper("OwnPrivate", data, forMapper, false),
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                const searchInput = data.searchInput as ResourceListSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "OwnPublic", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    OR: [
                        ...useVisibilityMapper("OwnPublic", data, forMapper, false),
                    ],
                };
            },
            public: function getPublic(data) {
                const searchInput = data.searchInput as ResourceListSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "Public", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    OR: [
                        ...useVisibilityMapper("Public", data, forMapper, false),
                    ],
                };
            },
        },
    }),
});
