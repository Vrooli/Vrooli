import { GqlModelType, MaxObjects, ResourceListFor, ResourceListSortBy, resourceListValidation, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel } from "../../builders/findFirstRel";
import { shapeHelper } from "../../builders/shapeHelper";
import { getLogic } from "../../getters/getLogic";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { ResourceListFormat } from "../formats";
import { ModelLogic } from "../types";
import { ApiVersionModel } from "./apiVersion";
import { FocusModeModel } from "./focusMode";
import { OrganizationModel } from "./organization";
import { PostModel } from "./post";
import { ProjectVersionModel } from "./projectVersion";
import { RoutineVersionModel } from "./routineVersion";
import { SmartContractVersionModel } from "./smartContractVersion";
import { StandardVersionModel } from "./standardVersion";
import { ResourceListModelLogic } from "./types";

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
const suppFields = [] as const;
export const ResourceListModel: ModelLogic<ResourceListModelLogic, typeof suppFields> = ({
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
            const { validate } = getLogic(["validate"], uppercaseFirstLetter(resourceOnType) as GqlModelType, ["en"], "ResourceListModel.validate.owner");
            return validate.owner(resourceOnData, userId);
        },
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ResourceListModelLogic["PrismaSelect"]>([
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
                    { apiVersion: ApiVersionModel.validate.visibility.owner(userId) },
                    { focusMode: FocusModeModel.validate.visibility.owner(userId) },
                    { organization: OrganizationModel.validate.visibility.owner(userId) },
                    { post: PostModel.validate.visibility.owner(userId) },
                    { projectVersion: ProjectVersionModel.validate.visibility.owner(userId) },
                    { routineVersion: RoutineVersionModel.validate.visibility.owner(userId) },
                    { smartContractVersion: SmartContractVersionModel.validate.visibility.owner(userId) },
                    { standardVersion: StandardVersionModel.validate.visibility.owner(userId) },
                ],
            }),
        },
    },
});
