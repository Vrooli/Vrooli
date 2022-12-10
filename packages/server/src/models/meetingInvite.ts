import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MeetingInvite } from "../endpoints/types";
import { PrismaType } from "../types";
import { MeetingModel } from "./meeting";
import { Displayer, Formatter } from "./types";

const __typename = 'MeetingInvite' as const;

const suppFields = [] as const;
const formatter = (): Formatter<MeetingInvite, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        meeting: 'Meeting',
        user: 'User',
    },
})

const displayer = (): Displayer<
    Prisma.meeting_inviteSelect,
    Prisma.meeting_inviteGetPayload<SelectWrap<Prisma.meeting_inviteSelect>>
> => ({
    select: () => ({ id: true, meeting: { select: MeetingModel.display.select() } }),
    // Label is the meeting label
    label: (select, languages) => MeetingModel.display.label(select.meeting as any, languages),
})

export const MeetingInviteModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})