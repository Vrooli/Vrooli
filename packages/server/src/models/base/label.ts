import { LabelSortBy, labelValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { LabelFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { LabelModelInfo, LabelModelLogic, OrganizationModelLogic } from "./types";

const __typename = "Label" as const;
export const LabelModel: LabelModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.label,
    display: () => ({
        label: {
            select: () => ({ id: true, label: true }),
            get: (select) => select.label,
        },
    }),
    format: LabelFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                label: data.label,
                color: noNull(data.color),
                ownedByOrganization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
                ownedByUser: !data.organizationConnect ? { connect: { id: rest.userData.id } } : undefined,
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                label: noNull(data.label),
                color: noNull(data.color),
                ...(await shapeHelper({ relation: "apis", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Api", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "focusModes", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "FocusMode", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "issues", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Issue", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "meetings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Meeting", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "notes", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Note", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "projects", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Project", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "routines", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Routine", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "schedules", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Schedule", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "smartContracts", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "SmartContract", parentRelationshipName: "label", data, ...rest })),
                ...(await shapeHelper({ relation: "standards", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Standard", parentRelationshipName: "label", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: labelValidation,
    },
    search: {
        defaultSort: LabelSortBy.DateUpdatedDesc,
        sortBy: LabelSortBy,
        searchFields: {
            createdTimeFrame: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "labelWrapped",
                "transDescriptionWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            ownedByOrganization: "Organization",
            ownedByUser: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data?.ownedByOrganization,
            User: data?.ownedByUser,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<LabelModelInfo["PrismaSelect"]>([
            ["ownedByOrganization", "Organization"],
            ["ownedByUser", "User"],
        ], ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId) },
                ],
            }),
        },
    }),
});
