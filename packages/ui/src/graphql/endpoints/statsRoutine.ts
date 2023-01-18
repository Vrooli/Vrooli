import { statsRoutineFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsRoutineEndpoint = {
    findMany: toQuery('statsRoutine', 'StatsRoutineSearchInput', toSearch(listFields)),
}