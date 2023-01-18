import { meetingInviteFields as fullFields, listMeetingInviteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const meetingInviteEndpoint = {
    findOne: toQuery('meetingInvite', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('meetingInvites', 'MeetingInviteSearchInput', toSearch(listFields)),
    create: toMutation('meetingInviteCreate', 'MeetingInviteCreateInput', fullFields[1]),
    update: toMutation('meetingInviteUpdate', 'MeetingInviteUpdateInput', fullFields[1]),
    accept: toMutation('meetingInviteAccept', 'FindByIdInput', fullFields[1]),
    decline: toMutation('meetingInviteDecline', 'FindByIdInput', fullFields[1])
}