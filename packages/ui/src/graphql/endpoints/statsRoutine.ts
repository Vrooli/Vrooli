import { statsRoutinePartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsRoutineEndpoint = {
    findMany: toQuery('statsRoutine', 'StatsRoutineSearchInput', ...toSearch(statsRoutinePartial)),
}