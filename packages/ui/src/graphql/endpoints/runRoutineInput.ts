import { runRoutineInputPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const runRoutineInputEndpoint = {
    findMany: toQuery('runRoutineInputs', 'RunRoutineInputSearchInput', ...toSearch(runRoutineInputPartial)),
}