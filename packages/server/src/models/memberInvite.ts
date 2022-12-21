import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MemberInvite, MemberInviteCreateInput, MemberInvitePermission, MemberInviteSearchInput, MemberInviteSortBy, MemberInviteUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = 'MemberInvite' as const;
const suppFields = [] as const;
export const MemberInviteModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MemberInviteCreateInput,
    GqlUpdate: MemberInviteUpdateInput,
    GqlModel: MemberInvite,
    GqlSearch: MemberInviteSearchInput,
    GqlSort: MemberInviteSortBy,
    GqlPermission: MemberInvitePermission,
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
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})