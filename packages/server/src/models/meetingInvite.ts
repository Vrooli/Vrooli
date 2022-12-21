import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MeetingInvite, MeetingInviteCreateInput, MeetingInvitePermission, MeetingInviteSearchInput, MeetingInviteSortBy, MeetingInviteUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { MeetingModel } from "./meeting";
import { ModelLogic } from "./types";

const __typename = 'MeetingInvite' as const;
const suppFields = [] as const;
export const MeetingInviteModel: ModelLogic<{
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
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: {
        select: () => ({ id: true, meeting: { select: MeetingModel.display.select() } }),
        // Label is the meeting label
        label: (select, languages) => MeetingModel.display.label(select.meeting as any, languages),
    },
    format: {
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
        countFields: {},
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})