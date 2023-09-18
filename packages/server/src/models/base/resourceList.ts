import { MaxObjects, ResourceListFor, ResourceListSortBy, resourceListValidation, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel, shapeHelper } from "../../builders";
import { getLogic } from "../../getters";
import { bestTranslation, defaultPermissions, oneIsPublic, translationShapeHelper } from "../../utils";
import { ResourceListFormat } from "../formats";
import { ModelLogic } from "../types";
import { ApiModel } from "./api";
import { FocusModeModel } from "./focusMode";
import { OrganizationModel } from "./organization";
import { PostModel } from "./post";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { SmartContractModel } from "./smartContract";
import { StandardModel } from "./standard";
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
                [forMapper[data.listFor]]: { connect: { id: data.listForConnect } },
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
            const { validate } = getLogic(["validate"], uppercaseFirstLetter(resourceOnType!) as any, ["en"], "ResourceListModel.validate.owner");
            return validate.owner(resourceOnData, userId);
        },
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.resource_listSelect>(data, [
            ["apiVersion", "Api"],
            ["focusMode", "FocusMode"],
            ["organization", "Organization"],
            ["post", "Post"],
            ["projectVersion", "Project"],
            ["routineVersion", "Routine"],
            ["smartContractVersion", "SmartContract"],
            ["standardVersion", "Standard"],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { apiVersion: ApiModel.validate.visibility.owner(userId) },
                    { focusMode: FocusModeModel.validate.visibility.owner(userId) },
                    { organization: OrganizationModel.validate.visibility.owner(userId) },
                    { post: PostModel.validate.visibility.owner(userId) },
                    { project: ProjectModel.validate.visibility.owner(userId) },
                    { routineVersion: RoutineModel.validate.visibility.owner(userId) },
                    { smartContractVersion: SmartContractModel.validate.visibility.owner(userId) },
                    { standardVersion: StandardModel.validate.visibility.owner(userId) },
                ],
            }),
        },
    },
});
