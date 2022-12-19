import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Meeting, MeetingCreateInput, MeetingSearchInput, MeetingSortBy, MeetingUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, Formatter, ModelLogic, Searcher } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MeetingCreateInput,
    GqlUpdate: MeetingUpdateInput,
    GqlModel: Meeting,
    GqlSearch: MeetingSearchInput,
    GqlSort: MeetingSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.meetingUpsertArgs['create'],
    PrismaUpdate: Prisma.meetingUpsertArgs['update'],
    PrismaModel: Prisma.meetingGetPayload<SelectWrap<Prisma.meetingSelect>>,
    PrismaSelect: Prisma.meetingSelect,
    PrismaWhere: Prisma.meetingWhereInput,
}

const __typename = 'Meeting' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        attendees: 'User',
        invites: 'MeetingInvite',
        labels: 'Label',
        organization: 'Organization',
        restrictedToRoles: 'Role',
    },
    prismaRelMap: {
        __typename,
        organization: 'Organization',
        restrictedToRoles: 'Role',
        attendees: 'User',
        invites: 'MeetingInvite',
        labels: 'Label',
    },
    joinMap: { 
        labels: 'label', 
        restrictedToRoles: 'role', 
        attendees: 'user',
        invites: 'user',
    },
    countFields: ['attendeesCount', 'invitesCount', 'labelsCount', 'translationsCount'],
})

const searcher = (): Searcher<Model> => ({
    defaultSort: MeetingSortBy.EventStartAsc,
    sortBy: MeetingSortBy,
    searchFields: [
        'createdTimeFrame',
        'labelsId',
        'openToAnyoneWithInvite',
        'organizationId',
        'showOnOrganizationProfile',
        'translationLanguages',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: () => ({
        OR: [
            'labelsWrapped',
            'transNameWrapped',
            'transDescriptionWrapped'
        ]
    })
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const MeetingModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: searcher(),
    validate: {} as any,
})