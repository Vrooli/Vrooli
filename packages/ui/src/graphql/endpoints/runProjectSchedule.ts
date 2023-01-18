import { runProjectScheduleFields as fullFields, listRunProjectScheduleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runProjectScheduleEndpoint = {
    findOne: toQuery('runProjectSchedule', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('runProjectSchedules', 'RunProjectScheduleSearchInput', toSearch(listFields)),
    create: toMutation('runProjectScheduleCreate', 'RunProjectScheduleCreateInput', fullFields[1]),
    update: toMutation('runProjectScheduleUpdate', 'RunProjectScheduleUpdateInput', fullFields[1])
}