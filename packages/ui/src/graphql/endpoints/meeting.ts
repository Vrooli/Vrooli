import { meetingFields as fullFields, listMeetingFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const meetingEndpoint = {
    findOne: toQuery('meeting', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('meetings', 'MeetingSearchInput', toSearch(listFields)),
    create: toMutation('meetingCreate', 'MeetingCreateInput', fullFields[1]),
    update: toMutation('meetingUpdate', 'MeetingUpdateInput', fullFields[1])
}