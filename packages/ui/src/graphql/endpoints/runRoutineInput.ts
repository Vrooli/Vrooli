import { runRoutineInputFields as fullFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const runRoutineInputEndpoint = {
    findMany: toQuery('runRoutineInputs', 'RunRoutineInputSearchInput', [fullFields], toSearch(fullFields)),
}