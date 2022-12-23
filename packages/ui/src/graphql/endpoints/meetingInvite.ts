import { meetingInviteFields as fullFields, listMeetingInviteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const meetingInviteEndpoint = {
    findOne: toQuery('meetingInvite', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('meetingInvites', 'MeetingInviteSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('meetingInviteCreate', 'MeetingInviteCreateInput', [fullFields], `...fullFields`),
    update: toMutation('meetingInviteUpdate', 'MeetingInviteUpdateInput', [fullFields], `...fullFields`),
    accept: toMutation('meetingInviteAccept', 'FindByIdInput', [fullFields], `...fullFields`),
    decline: toMutation('meetingInviteDecline', 'FindByIdInput', [fullFields], `...fullFields`)
}