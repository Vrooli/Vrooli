import { runRoutineSchedulePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const runRoutineScheduleEndpoint = {
    findOne: toQuery('runRoutineSchedule', 'FindByIdInput', runRoutineSchedulePartial, 'full'),
    findMany: toQuery('runRoutineSchedules', 'RunRoutineScheduleSearchInput', ...toSearch(runRoutineSchedulePartial)),
    create: toMutation('runRoutineScheduleCreate', 'RunRoutineScheduleCreateInput', runRoutineSchedulePartial, 'full'),
    update: toMutation('runRoutineScheduleUpdate', 'RunRoutineScheduleUpdateInput', runRoutineSchedulePartial, 'full')
}