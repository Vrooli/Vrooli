import { MaxObjects, RoleSortBy, roleValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../../utils";
import { RoleFormat } from "../format/role";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { RoleModelLogic } from "./types";

const __typename = "Role" as const;
const suppFields = [] as const;
export const RoleModel: ModelLogic<RoleModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.role,
    display: {
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
    },
    format: RoleFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                permissions: data.permissions,
                ...(await shapeHelper({ relation: "members", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "roles", data, ...rest })),
                ...(await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Organization", parentRelationshipName: "roles", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                permissions: noNull(data.permissions),
                ...(await shapeHelper({ relation: "members", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Member", parentRelationshipName: "roles", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
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
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, organization: "Organization" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => OrganizationModel.validate.owner(data.organization as any, userId),
        isDeleted: (data, languages) => OrganizationModel.validate.isDeleted(data.organization as any, languages),
        isPublic: (data, languages) => OrganizationModel.validate.isPublic(data.organization as any, languages),
        visibility: {
            private: { organization: OrganizationModel.validate.visibility.private },
            public: { organization: OrganizationModel.validate.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate.visibility.owner(userId) }),
        },
    },
});
