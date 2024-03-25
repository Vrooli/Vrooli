import { MaxObjects, RoleSortBy, roleValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { RoleFormat } from "../formats";
import { OrganizationModelInfo, OrganizationModelLogic, RoleModelInfo, RoleModelLogic } from "./types";

const __typename = "Role" as const;
export const RoleModel: RoleModelLogic = ({
    __typename,
    delegate: (p) => p.role,
    display: () => ({
        label: {
            select: () => ({
                id: true,
                name: true,
                translations: { select: { language: true, name: true } },
            }),
            get: (select, languages) => {
                // Prefer translated name over default name
                const translated = bestTranslation(select.translations, languages)?.name ?? "";
                if (translated.length > 0) return translated;
                return select.name;
            },
        },
    }),
    format: RoleFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                permissions: data.permissions,
                members: await shapeHelper({ relation: "members", relTypes: ["Connect"], isOneToOne: false, objectType: "Member", parentRelationshipName: "roles", data, ...rest }),
                organization: await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, objectType: "Organization", parentRelationshipName: "roles", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                permissions: noNull(data.permissions),
                members: await shapeHelper({ relation: "members", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Member", parentRelationshipName: "roles", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: roleValidation,
    },
    search: {
        defaultSort: RoleSortBy.MembersDesc,
        sortBy: RoleSortBy,
        searchFields: {
            createdTimeFrame: true,
            translationLanguages: true,
            organizationId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                "transDescriptionWrapped",
                "transNameWrapped",
            ],
        }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, organization: "Organization" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<OrganizationModelLogic>("Organization").validate().owner(data?.organization as OrganizationModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<OrganizationModelLogic>("Organization").validate().isDeleted(data.organization as OrganizationModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<RoleModelInfo["PrismaSelect"]>([["organization", "Organization"]], ...rest),
        visibility: {
            private: { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate().visibility.private },
            public: { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate().visibility.public },
            owner: (userId) => ({ organization: ModelMap.get<OrganizationModelLogic>("Organization").validate().visibility.owner(userId) }),
        },
    }),
});
