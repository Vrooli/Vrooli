import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Meeting } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, Formatter } from "./types";

const __typename = 'Meeting' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Meeting, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        attendees: 'User',
        invites: 'MeetingInvite',
        labels: 'Label',
        organization: 'Organization',
        restrictedToRoles: 'Role',
    },
    joinMap: { labels: 'label' },
    countFields: ['attendeesCount', 'invitesCount', 'labelsCount', 'translationsCount'],
})

const displayer = (): Displayer<
    Prisma.meetingSelect,
    Prisma.meetingGetPayload<SelectWrap<Prisma.meetingSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const MeetingModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})