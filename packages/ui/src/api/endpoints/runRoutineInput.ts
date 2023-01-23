import { runRoutineInputPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const runRoutineInputEndpoint = {
    findMany: toQuery('runRoutineInputs', 'RunRoutineInputSearchInput', ...toSearch(runRoutineInputPartial)),
}