import { runRoutineFields as fullFields, listRunRoutineFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runRoutineEndpoint = {
    findOne: toQuery('runRoutine', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('runRoutines', 'RunRoutineSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('runRoutineCreate', 'RunRoutineCreateInput', [fullFields], `...fullFields`),
    update: toMutation('runRoutineUpdate', 'RunRoutineUpdateInput', [fullFields], `...fullFields`),
    deleteAll: toMutation('runRoutineDeleteAll', null, [], `count`),
    complete: toMutation('runRoutineComplete', 'RunRoutineCompleteInput', [fullFields], `...fullFields`),
    cancel: toMutation('runRoutineCancel', 'RunRoutineCancelInput', [fullFields], `...fullFields`),
}