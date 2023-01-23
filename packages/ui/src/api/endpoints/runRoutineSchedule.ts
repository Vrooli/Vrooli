import { runRoutineSchedulePartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const runRoutineScheduleEndpoint = {
    findOne: toQuery('runRoutineSchedule', 'FindByIdInput', runRoutineSchedulePartial, 'full'),
    findMany: toQuery('runRoutineSchedules', 'RunRoutineScheduleSearchInput', ...toSearch(runRoutineSchedulePartial)),
    create: toMutation('runRoutineScheduleCreate', 'RunRoutineScheduleCreateInput', runRoutineSchedulePartial, 'full'),
    update: toMutation('runRoutineScheduleUpdate', 'RunRoutineScheduleUpdateInput', runRoutineSchedulePartial, 'full')
}