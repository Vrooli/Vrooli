import { runProjectScheduleFields as fullFields, listRunProjectScheduleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runProjectScheduleEndpoint = {
    findOne: toQuery('runProjectSchedule', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('runProjectSchedules', 'RunProjectScheduleSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('runProjectScheduleCreate', 'RunProjectScheduleCreateInput', [fullFields], `...fullFields`),
    update: toMutation('runProjectScheduleUpdate', 'RunProjectScheduleUpdateInput', [fullFields], `...fullFields`)
}