import { runRoutineFields as fullFields, listRunRoutineFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runRoutineEndpoint = {
    findOne: toQuery('runRoutine', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('runRoutines', 'RunRoutineSearchInput', toSearch(listFields)),
    create: toMutation('runRoutineCreate', 'RunRoutineCreateInput', fullFields[1]),
    update: toMutation('runRoutineUpdate', 'RunRoutineUpdateInput', fullFields[1]),
    deleteAll: toMutation('runRoutineDeleteAll', null, `{ count }`),
    complete: toMutation('runRoutineComplete', 'RunRoutineCompleteInput', fullFields[1]),
    cancel: toMutation('runRoutineCancel', 'RunRoutineCancelInput', fullFields[1]),
}