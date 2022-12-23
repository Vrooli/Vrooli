import { userScheduleFields as fullFields, listUserScheduleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const userScheduleEndpoint = {
    findOne: toQuery('userSchedule', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('userSchedules', 'UserScheduleSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('userScheduleCreate', 'UserScheduleCreateInput', [fullFields], `...fullFields`),
    update: toMutation('userScheduleUpdate', 'UserScheduleUpdateInput', [fullFields], `...fullFields`)
}