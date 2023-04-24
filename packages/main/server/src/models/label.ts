import { Label, LabelCreateInput, LabelSearchInput, LabelSortBy, LabelUpdateInput, LabelYou, MaxObjects } from ":local/consts";
import { labelValidation } from ":local/validation";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic, translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";

const __typename = "Label" as const;
type Permissions = Pick<LabelYou, "canDelete" | "canUpdate">;
const suppFields = ["you"] as const;
export const LabelModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: LabelCreateInput,
    GqlUpdate: LabelUpdateInput,
    GqlModel: Label,
    GqlSearch: LabelSearchInput,
    GqlSort: LabelSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.labelUpsertArgs["create"],
    PrismaUpdate: Prisma.labelUpsertArgs["update"],
    PrismaModel: Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>,
    PrismaSelect: Prisma.labelSelect,
    PrismaWhere: Prisma.labelWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.label,
    display: {
        select: () => ({ id: true, label: true }),
        label: (select) => select.label,
    },
    format: {
        gqlRelMap: {
            __typename,
            apis: "Api",
            focusModes: "FocusMode",
            issues: "Issue",
            meetings: "Meeting",
            notes: "Note",
            owner: {
                ownedByUser: "User",
                ownedByOrganization: "Organization",
            },
            projects: "Project",
            routines: "Routine",
            schedules: "Schedule",
        },
        prismaRelMap: {
            __typename,
            apis: "Api",
            focusModes: "FocusMode",
            issues: "Issue",
            meetings: "Meeting",
            notes: "Note",
            ownedByUser: "User",
            ownedByOrganization: "Organization",
            projects: "Project",
            routines: "Routine",
            schedules: "Schedule",
        },
        joinMap: {
            apis: "labelled",
            focusModes: "labelled",
            issues: "labelled",
            meetings: "labelled",
            notes: "labelled",
            projects: "labelled",
            routines: "labelled",
            schedules: "labelled",
        },
        countFields: {
            apisCount: true,
            focusModesCount: true,
            issuesCount: true,
            meetingsCount: true,
            notesCount: true,
            projectsCount: true,
            smartContractsCount: true,
            standardsCount: true,
            routinesCount: true,
            schedulesCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "labelWrapped",
                "transDescriptionWrapped",
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            ownedByOrganization: "Organization",
            ownedByUser: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.commentSelect>(data, [
            ["ownedByOrganization", "Organization"],
            ["ownedByUser", "User"],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
