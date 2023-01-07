import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MemberInvite, MemberInviteCreateInput, MemberInviteSearchInput, MemberInviteSortBy, MemberInviteUpdateInput, MemberInviteYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { UserModel } from "./user";
import { getSingleTypePermissions } from "../validators";

const type = 'MemberInvite' as const;
type Permissions = Pick<MemberInviteYou, 'canDelete' | 'canEdit'>;
const suppFields = ['you.canDelete', 'you.canEdit'] as const;
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
    type,
    delegate: (prisma: PrismaType) => prisma.member_invite,
    display: {
        select: () => ({ id: true, user: { select: UserModel.display.select() } }),
        // Label is the member label
        label: (select, languages) => UserModel.display.label(select.user as any, languages),
    },
    format: {
        gqlRelMap: {
            type,
            organization: 'Organization',
            user: 'User',
        },
        prismaRelMap: {
            type,
            organization: 'Organization',
            user: 'User',
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})