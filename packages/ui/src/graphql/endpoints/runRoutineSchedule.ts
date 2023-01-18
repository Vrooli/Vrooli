import { runRoutineScheduleFields as fullFields, listRunRoutineScheduleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runRoutineScheduleEndpoint = {
    findOne: toQuery('runRoutineSchedule', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('runRoutineSchedules', 'RunRoutineScheduleSearchInput', toSearch(listFields)),
    create: toMutation('runRoutineScheduleCreate', 'RunRoutineScheduleCreateInput', fullFields[1]),
    update: toMutation('runRoutineScheduleUpdate', 'RunRoutineScheduleUpdateInput', fullFields[1])
}