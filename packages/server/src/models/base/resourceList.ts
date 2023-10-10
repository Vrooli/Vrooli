import { GqlModelType, MaxObjects, ResourceListFor, ResourceListSortBy, resourceListValidation, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { ResourceListFormat } from "../formats";
import { ApiVersionModelLogic, FocusModeModelLogic, OrganizationModelLogic, PostModelLogic, ProjectVersionModelLogic, ResourceListModelInfo, ResourceListModelLogic, RoutineVersionModelLogic, SmartContractVersionModelLogic, StandardVersionModelLogic } from "./types";

const forMapper: { [key in ResourceListFor]: keyof Prisma.resource_listUpsertArgs["create"] } = {
    ApiVersion: "apiVersion",
    FocusMode: "focusMode",
    Organization: "organization",
    Post: "post",
    ProjectVersion: "projectVersion",
    RoutineVersion: "routineVersion",
    SmartContractVersion: "smartContractVersion",
    StandardVersion: "standardVersion",
};

const __typename = "ResourceList" as const;
export const ResourceListModel: ResourceListModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.resource_list,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
    },
    format: ResourceListFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                [forMapper[data.listForType]]: { connect: { id: data.listForConnect } },
                ...(await shapeHelper({ relation: "resources", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Resource", parentRelationshipName: "list", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await shapeHelper({ relation: "resources", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "Resource", parentRelationshipName: "list", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: resourceListValidation,
    },
    search: {
        defaultSort: ResourceListSortBy.IndexAsc,
        sortBy: ResourceListSortBy,
        searchFields: {
            apiVersionId: true,
            createdTimeFrame: true,
            organizationId: true,
            postId: true,
            projectVersionId: true,
            routineVersionId: true,
            smartContractVersionId: true,
            standardVersionId: true,
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
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            apiVersion: "ApiVersion",
            organization: "Organization",
            post: "Post",
            projectVersion: "ProjectVersion",
            routineVersion: "RoutineVersion",
            smartContractVersion: "SmartContractVersion",
            standardVersion: "StandardVersion",
            focusMode: "FocusMode",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => {
            if (!data) return {};
            const [resourceOnType, resourceOnData] = findFirstRel(data, [
                "apiVersion",
                "focusMode",
                "organization",
                "post",
                "projectVersion",
                "routineVersion",
                "smartContractVersion",
                "standardVersion",
            ]);
            if (!resourceOnType || !resourceOnData) return {};
            const validate = ModelMap.get(uppercaseFirstLetter(resourceOnType) as GqlModelType).validate;
            return validate.owner(resourceOnData, userId);
        },
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceListModelInfo["PrismaSelect"]>([
            ["apiVersion", "ApiVersion"],
            ["focusMode", "FocusMode"],
            ["organization", "Organization"],
            ["post", "Post"],
            ["projectVersion", "ProjectVersion"],
            ["routineVersion", "RoutineVersion"],
            ["smartContractVersion", "SmartContractVersion"],
            ["standardVersion", "StandardVersion"],
        ], ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { apiVersion: ModelMap.get<ApiVersionModelLogic>("ApiVersion").validate.visibility.owner(userId) },
                    { focusMode: ModelMap.get<FocusModeModelLogic>("FocusMode").validate.visibility.owner(userId) },
                    { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate.visibility.owner(userId) },
                    { post: ModelMap.get<PostModelLogic>("Post").validate.visibility.owner(userId) },
                    { projectVersion: ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").validate.visibility.owner(userId) },
                    { routineVersion: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate.visibility.owner(userId) },
                    { smartContractVersion: ModelMap.get<SmartContractVersionModelLogic>("SmartContractVersion").validate.visibility.owner(userId) },
                    { standardVersion: ModelMap.get<StandardVersionModelLogic>("StandardVersion").validate.visibility.owner(userId) },
                ],
            }),
        },
    },
});
