import { meetingPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const meetingEndpoint = {
    findOne: toQuery('meeting', 'FindByIdInput', meetingPartial, 'full'),
    findMany: toQuery('meetings', 'MeetingSearchInput', ...toSearch(meetingPartial)),
    create: toMutation('meetingCreate', 'MeetingCreateInput', meetingPartial, 'full'),
    update: toMutation('meetingUpdate', 'MeetingUpdateInput', meetingPartial, 'full')
}