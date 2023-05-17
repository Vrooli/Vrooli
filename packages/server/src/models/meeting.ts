import { MaxObjects, Meeting, MeetingCreateInput, MeetingSearchInput, MeetingSortBy, MeetingUpdateInput, meetingValidation, MeetingYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, labelShapeHelper, onCommonPlain, translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";

const __typename = "Meeting" as const;
type Permissions = Pick<MeetingYou, "canDelete" | "canInvite" | "canUpdate">;
const suppFields = ["you"] as const;
export const MeetingModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MeetingCreateInput,
    GqlUpdate: MeetingUpdateInput,
    GqlModel: Meeting,
    GqlSearch: MeetingSearchInput,
    GqlSort: MeetingSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.meetingUpsertArgs["create"],
    PrismaUpdate: Prisma.meetingUpsertArgs["update"],
    PrismaModel: Prisma.meetingGetPayload<SelectWrap<Prisma.meetingSelect>>,
    PrismaSelect: Prisma.meetingSelect,
    PrismaWhere: Prisma.meetingWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return JSON.stringify({
                    description: trans.description,
                    name: trans.name,
                });
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            attendees: "User",
            invites: "MeetingInvite",
            labels: "Label",
            organization: "Organization",
            restrictedToRoles: "Role",
            schedule: "Schedule",
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            restrictedToRoles: "Role",
            attendees: "User",
            invites: "MeetingInvite",
            labels: "Label",
            schedule: "Schedule",
        },
        joinMap: {
            labels: "label",
            restrictedToRoles: "role",
            attendees: "user",
            invites: "user",
        },
        countFields: {
            attendeesCount: true,
            invitesCount: true,
            labelsCount: true,
            translationsCount: true,
        },
        supplemental: {
            dbFields: ["organizationId"],
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, prisma, userData }) => {
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
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                showOnOrganizationProfile: noNull(data.showOnOrganizationProfile),
                ...(await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Organization", parentRelationshipName: "meetings", data, ...rest })),
                ...(await shapeHelper({
                    relation: "restrictedToRoles", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "", joinData: {
                        fieldName: "role",
                        uniqueFieldName: "meeting_roles_meetingid_roleid_unique",
                        childIdFieldName: "roleId",
                        parentIdFieldName: "meetingId",
                        parentId: data.id ?? null,
                    }, data, ...rest,
                })),
                ...(await shapeHelper({ relation: "invites", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "MeetingInvite", parentRelationshipName: "meeting", data, ...rest })),
                ...(await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "Schedule", parentRelationshipName: "meetings", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Meeting", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                showOnOrganizationProfile: noNull(data.showOnOrganizationProfile),
                ...(await shapeHelper({
                    relation: "restrictedToRoles", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "", joinData: {
                        fieldName: "role",
                        uniqueFieldName: "meeting_roles_meetingid_roleid_unique",
                        childIdFieldName: "roleId",
                        parentIdFieldName: "meetingId",
                        parentId: data.id ?? null,
                    }, data, ...rest,
                })),
                ...(await shapeHelper({ relation: "invites", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "MeetingInvite", parentRelationshipName: "meeting", data, ...rest })),
                ...(await shapeHelper({ relation: "schedule", relTypes: ["Create", "Connect", "Update", "Delete"], isOneToOne: true, isRequired: false, objectType: "Schedule", parentRelationshipName: "meetings", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Create", "Update"], parentType: "Meeting", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonPlain({
                    ...params,
                    objectType: __typename,
                    ownerOrganizationField: "organization",
                });
            },
        },
        yup: meetingValidation,
    },
    search: {
        defaultSort: MeetingSortBy.AttendeesDesc,
        sortBy: MeetingSortBy,
        searchFields: {
            createdTimeFrame: true,
            labelsIds: true,
            openToAnyoneWithInvite: true,
            organizationId: true,
            scheduleEndTimeFrame: true,
            scheduleStartTimeFrame: true,
            showOnOrganizationProfile: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "labelsWrapped",
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            showOnOrganizationProfile: true,
            organization: "Organization",
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canInvite: () => isLoggedIn && isAdmin,
        }),
        owner: (data) => ({
            Organization: data.organization,
        }),
        isDeleted: () => false,
        isPublic: (data) => data.showOnOrganizationProfile === true,
        visibility: {
            private: {
                OR: [
                    { showOnOrganizationProfile: false },
                    { organization: { isPrivate: true } },
                ],
            },
            public: {
                AND: [
                    { showOnOrganizationProfile: true },
                    { organization: { isPrivate: false } },
                ],
            },
            owner: (userId) => ({
                organization: OrganizationModel.query.hasRoleQuery(userId),
            }),
        },
    },
});
