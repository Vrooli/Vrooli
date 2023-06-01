import { Role, RoleCreateInput, RoleSearchInput, RoleSortBy, RoleUpdateInput, roleValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { bestTranslation, translationShapeHelper } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Role" as const;
export const RoleFormat: Formatter<ModelRoleLogic> = {
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
        searchFields: {
            createdTimeFrame: true,
            translationLanguages: true,
            organizationId: true,
            updatedTimeFrame: true,
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                "transDescriptionWrapped",
                "transNameWrapped",
            ],
};
