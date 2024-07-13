import { GqlModelType, MaxObjects, ResourceListFor, ResourceListSortBy, resourceListValidation, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
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

const __typename = "ResourceList" as const;
export const ResourceListModel: ResourceListModelLogic = ({
    __typename,
    dbTable: "resource_list",
    dbTranslationTable: "resource_list_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
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
            ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value, key as GqlModelType])),
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => {
            if (!data) return {};
            const [resourceOnType, resourceOnData] = findFirstRel(data, Object.values(forMapper));
            if (!resourceOnType || !resourceOnData) return {};
            return ModelMap.get(uppercaseFirstLetter(resourceOnType) as GqlModelType).validate().owner(resourceOnData, userId);
        },
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceListModelInfo["PrismaSelect"]>(Object.entries(forMapper).map(([key, value]) => [value, key as GqlModelType]), ...rest),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({
                OR: [
                    ...Object.entries(forMapper).map(([key, value]) => ({ [value]: ModelMap.get(key as GqlModelType).validate().visibility.owner(userId) })),
                ],
            }),
        },
    }),
});
