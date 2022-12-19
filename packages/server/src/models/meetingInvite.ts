import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MeetingInvite, MeetingInviteCreateInput, MeetingInvitePermission, MeetingInviteSearchInput, MeetingInviteSortBy, MeetingInviteUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { MeetingModel } from "./meeting";
import { Displayer, Formatter, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MeetingInviteCreateInput,
    GqlUpdate: MeetingInviteUpdateInput,
    GqlModel: MeetingInvite,
    GqlSearch: MeetingInviteSearchInput,
    GqlSort: MeetingInviteSortBy,
    GqlPermission: MeetingInvitePermission,
    PrismaCreate: Prisma.meeting_inviteUpsertArgs['create'],
    PrismaUpdate: Prisma.meeting_inviteUpsertArgs['update'],
    PrismaModel: Prisma.meeting_inviteGetPayload<SelectWrap<Prisma.meeting_inviteSelect>>,
    PrismaSelect: Prisma.meeting_inviteSelect,
    PrismaWhere: Prisma.meeting_inviteWhereInput,
}

const __typename = 'MeetingInvite' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        meeting: 'Meeting',
        user: 'User',
    },
    prismaRelMap: {
        __typename,
        meeting: 'Meeting',
        user: 'User',
    },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, meeting: { select: MeetingModel.display.select() } }),
    // Label is the meeting label
    label: (select, languages) => MeetingModel.display.label(select.meeting as any, languages),
})

export const MeetingInviteModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})