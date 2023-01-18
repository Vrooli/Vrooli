import { userScheduleFields as fullFields, listUserScheduleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const userScheduleEndpoint = {
    findOne: toQuery('userSchedule', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('userSchedules', 'UserScheduleSearchInput', toSearch(listFields)),
    create: toMutation('userScheduleCreate', 'UserScheduleCreateInput', fullFields[1]),
    update: toMutation('userScheduleUpdate', 'UserScheduleUpdateInput', fullFields[1])
}