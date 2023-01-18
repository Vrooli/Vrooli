import { routineFields as fullFields, listRoutineFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const routineEndpoint = {
    findOne: toQuery('routine', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('routines', 'RoutineSearchInput', toSearch(listFields)),
    create: toMutation('routineCreate', 'RoutineCreateInput', fullFields[1]),
    update: toMutation('routineUpdate', 'RoutineUpdateInput', fullFields[1])
}