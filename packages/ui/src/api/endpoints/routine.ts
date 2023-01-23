import { routinePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const routineEndpoint = {
    findOne: toQuery('routine', 'FindByIdInput', routinePartial, 'full'),
    findMany: toQuery('routines', 'RoutineSearchInput', ...toSearch(routinePartial)),
    create: toMutation('routineCreate', 'RoutineCreateInput', routinePartial, 'full'),
    update: toMutation('routineUpdate', 'RoutineUpdateInput', routinePartial, 'full')
}