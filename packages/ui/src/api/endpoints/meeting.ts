import { meetingPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const meetingEndpoint = {
    findOne: toQuery('meeting', 'FindByIdInput', meetingPartial, 'full'),
    findMany: toQuery('meetings', 'MeetingSearchInput', ...toSearch(meetingPartial)),
    create: toMutation('meetingCreate', 'MeetingCreateInput', meetingPartial, 'full'),
    update: toMutation('meetingUpdate', 'MeetingUpdateInput', meetingPartial, 'full')
}