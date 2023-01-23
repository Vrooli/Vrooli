import { countPartial, runRoutinePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const runRoutineEndpoint = {
    findOne: toQuery('runRoutine', 'FindByIdInput', runRoutinePartial, 'full'),
    findMany: toQuery('runRoutines', 'RunRoutineSearchInput', ...toSearch(runRoutinePartial)),
    create: toMutation('runRoutineCreate', 'RunRoutineCreateInput', runRoutinePartial, 'full'),
    update: toMutation('runRoutineUpdate', 'RunRoutineUpdateInput', runRoutinePartial, 'full'),
    deleteAll: toMutation('runRoutineDeleteAll', null, countPartial, 'full'),
    complete: toMutation('runRoutineComplete', 'RunRoutineCompleteInput', runRoutinePartial, 'full'),
    cancel: toMutation('runRoutineCancel', 'RunRoutineCancelInput', runRoutinePartial, 'full'),
}