import { MaxObjects, MemberInviteSortBy, memberInviteValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MemberInviteFormat } from "../formats";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { MemberInviteModelLogic, UserModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "MemberInvite" as const;
const suppFields = ["you"] as const;
export const MemberInviteModel: ModelLogic<MemberInviteModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.member_invite,
    display: {
        // Label is the member label
        label: {
            select: () => ({ id: true, user: { select: UserModel.display.label.select() } }),
            get: (select, languages) => UserModel.display.label.get(select.user as UserModelLogic["PrismaModel"], languages),
        },
    },
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
                { organization: OrganizationModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
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
    validate: {
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
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.member_inviteSelect>(data, [
            ["organization", "Organization"],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                organization: OrganizationModel.query.hasRoleQuery(userId),
            }),
        },
    },
});
