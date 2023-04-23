import { RoleSortBy } from "@local/consts";
import { roleValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { bestLabel, translationShapeHelper } from "../utils";
const __typename = "Role";
const suppFields = [];
export const RoleModel = ({
    __typename,
    delegate: (prisma) => prisma.role,
    display: {
        select: () => ({
            id: true,
            name: true,
            translations: { select: { language: true, name: true } },
        }),
        label: (select, languages) => {
            const translated = bestLabel(select.translations, "name", languages);
            if (translated.length > 0)
                return translated;
            return select.name;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            members: "Member",
            organization: "Organization",
        },
        prismaRelMap: {
            __typename,
            members: "Member",
            meetings: "Meeting",
            organization: "Organization",
        },
        countFields: {
            membersCount: true,
        },
    },
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                "transDescriptionWrapped",
                "transNameWrapped",
            ],
        }),
    },
    validate: {},
});
//# sourceMappingURL=role.js.map