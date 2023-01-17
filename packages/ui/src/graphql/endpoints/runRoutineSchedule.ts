import { runRoutineScheduleFields as fullFields, listRunRoutineScheduleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runRoutineScheduleEndpoint = {
    findOne: toQuery('runRoutineSchedule', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('runRoutineSchedules', 'RunRoutineScheduleSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('runRoutineScheduleCreate', 'RunRoutineScheduleCreateInput', [fullFields], `...fullFields`),
    update: toMutation('runRoutineScheduleUpdate', 'RunRoutineScheduleUpdateInput', [fullFields], `...fullFields`)
}