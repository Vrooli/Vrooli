import { runRoutineInputPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const runRoutineInputEndpoint = {
    findMany: toQuery('runRoutineInputs', 'RunRoutineInputSearchInput', ...toSearch(runRoutineInputPartial)),
}