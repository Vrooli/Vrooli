import { MaxObjects, MemberInviteSortBy, memberInviteValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MemberInviteFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MemberInviteModelInfo, MemberInviteModelLogic, OrganizationModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "MemberInvite" as const;
export const MemberInviteModel: MemberInviteModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.member_invite,
    display: () => ({
        // Label is the member label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["PrismaModel"], languages),
        },
    }),
    format: MemberInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                message: noNull(data.message),
                willBeAdmin: noNull(data.willBeAdmin),
                willHavePermissions: noNull(data.willHavePermissions),
                ...(await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Organization", parentRelationshipName: "memberInvites", data, ...rest })),
                ...(await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "User", parentRelationshipName: "membershipsInvited", data, ...rest })),
            }),
            update: async ({ data }) => ({
                message: noNull(data.message),
                willBeAdmin: noNull(data.willBeAdmin),
                willHavePermissions: noNull(data.willHavePermissions),
            }),
        },
        yup: memberInviteValidation,
    },
    search: {
        defaultSort: MemberInviteSortBy.DateUpdatedDesc,
        sortBy: MemberInviteSortBy,
        searchFields: {
            createdTimeFrame: true,
            organizationId: true,
            status: true,
            statuses: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { organization: ModelMap.get<OrganizationModelLogic>("Organization").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
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
            isAdmin: true,
            permissions: true,
            organization: "Organization",
            user: "User",
            roles: "Role",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data?.organization,
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MemberInviteModelInfo["PrismaSelect"]>([["organization", "Organization"]], ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                organization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId),
            }),
        },
    }),
});
