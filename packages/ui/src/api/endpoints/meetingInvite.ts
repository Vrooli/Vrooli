import { meetingInvitePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const meetingInviteEndpoint = {
    findOne: toQuery('meetingInvite', 'FindByIdInput', meetingInvitePartial, 'full'),
    findMany: toQuery('meetingInvites', 'MeetingInviteSearchInput', ...toSearch(meetingInvitePartial)),
    create: toMutation('meetingInviteCreate', 'MeetingInviteCreateInput', meetingInvitePartial, 'full'),
    update: toMutation('meetingInviteUpdate', 'MeetingInviteUpdateInput', meetingInvitePartial, 'full'),
    accept: toMutation('meetingInviteAccept', 'FindByIdInput', meetingInvitePartial, 'full'),
    decline: toMutation('meetingInviteDecline', 'FindByIdInput', meetingInvitePartial, 'full')
}