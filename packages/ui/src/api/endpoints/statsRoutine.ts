import { statsRoutinePartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsRoutineEndpoint = {
    findMany: toQuery('statsRoutine', 'StatsRoutineSearchInput', ...toSearch(statsRoutinePartial)),
}