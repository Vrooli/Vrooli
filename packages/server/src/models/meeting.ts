import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Meeting, MeetingCreateInput, MeetingSearchInput, MeetingSortBy, MeetingUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const type = 'Meeting' as const;
const suppFields = [] as const;
export const MeetingModel: ModelLogic<{
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
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            type,
            attendees: 'User',
            invites: 'MeetingInvite',
            labels: 'Label',
            organization: 'Organization',
            restrictedToRoles: 'Role',
        },
        prismaRelMap: {
            type,
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
        countFields: {
            attendeesCount: true,
            invitesCount: true,
            labelsCount: true,
            translationsCount: true,
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: MeetingSortBy.EventStartAsc,
        sortBy: MeetingSortBy,
        searchFields: {
            createdTimeFrame: true,
            labelsId: true,
            openToAnyoneWithInvite: true,
            organizationId: true,
            showOnOrganizationProfile: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'labelsWrapped',
                'transNameWrapped',
                'transDescriptionWrapped'
            ]
        })
    },
    validate: {} as any,
})