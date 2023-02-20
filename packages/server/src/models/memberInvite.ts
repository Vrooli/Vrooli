import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, MemberInvite, MemberInviteCreateInput, MemberInviteSearchInput, MemberInviteSortBy, MemberInviteUpdateInput, MemberInviteYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { UserModel } from "./user";
import { getSingleTypePermissions } from "../validators";
import { defaultPermissions, oneIsPublic } from "../utils";
import { OrganizationModel } from "./organization";

const __typename = 'MemberInvite' as const;
type Permissions = Pick<MemberInviteYou, 'canDelete' | 'canUpdate'>;
const suppFields = ['you'] as const;
export const MemberInviteModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MemberInviteCreateInput,
    GqlUpdate: MemberInviteUpdateInput,
    GqlModel: MemberInvite,
    GqlSearch: MemberInviteSearchInput,
    GqlSort: MemberInviteSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.member_inviteUpsertArgs['create'],
    PrismaUpdate: Prisma.member_inviteUpsertArgs['update'],
    PrismaModel: Prisma.member_inviteGetPayload<SelectWrap<Prisma.member_inviteSelect>>,
    PrismaSelect: Prisma.member_inviteSelect,
    PrismaWhere: Prisma.member_inviteWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.member_invite,
    display: {
        select: () => ({ id: true, user: { select: UserModel.display.select() } }),
        // Label is the member label
        label: (select, languages) => UserModel.display.label(select.user as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
            },
        },
    },
    mutate: {} as any,
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
                'messageWrapped',
                { organization: OrganizationModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
            ]
        })
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            isAdmin: true,
            permissions: true,
            organization: 'Organization',
            user: 'User',
            roles: 'Role',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.member_inviteSelect>(data, [
            ['organization', 'Organization'],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                organization: OrganizationModel.query.hasRoleQuery(userId),
            }),
        }
    },
})