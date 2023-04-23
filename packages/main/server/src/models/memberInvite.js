import { MaxObjects, MemberInviteSortBy } from "@local/consts";
import { memberInviteValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
import { UserModel } from "./user";
const __typename = "MemberInvite";
const suppFields = ["you"];
export const MemberInviteModel = ({
    __typename,
    delegate: (prisma) => prisma.member_invite,
    display: {
        select: () => ({ id: true, user: { select: UserModel.display.select() } }),
        label: (select, languages) => UserModel.display.label(select.user, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: "Organization",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            user: "User",
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
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
            updatedTimeFrame: true,
            userId: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { organization: OrganizationModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
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
        isPublic: (data, languages) => oneIsPublic(data, [
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
//# sourceMappingURL=memberInvite.js.map