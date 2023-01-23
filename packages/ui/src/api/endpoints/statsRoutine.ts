import { statsRoutinePartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsRoutineEndpoint = {
    findMany: toQuery('statsRoutine', 'StatsRoutineSearchInput', ...toSearch(statsRoutinePartial)),
}