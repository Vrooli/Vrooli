import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MemberInvite, MemberInviteCreateInput, MemberInvitePermission, MemberInviteSearchInput, MemberInviteSortBy, MemberInviteUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";
import { UserModel } from "./user";

type Model = {
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
}


const __typename = 'MemberInvite' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, user: { select: UserModel.display.select() } }),
    // Label is the member label
    label: (select, languages) => UserModel.display.label(select.user as any, languages),
})

export const MemberInviteModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.member_invite,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})