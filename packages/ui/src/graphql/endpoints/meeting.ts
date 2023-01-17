import { meetingFields as fullFields, listMeetingFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const meetingEndpoint = {
    findOne: toQuery('meeting', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('meetings', 'MeetingSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('meetingCreate', 'MeetingCreateInput', [fullFields], `...fullFields`),
    update: toMutation('meetingUpdate', 'MeetingUpdateInput', [fullFields], `...fullFields`)
}