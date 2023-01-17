import { routineFields as fullFields, listRoutineFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const routineEndpoint = {
    findOne: toQuery('routine', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('routines', 'RoutineSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('routineCreate', 'RoutineCreateInput', [fullFields], `...fullFields`),
    update: toMutation('routineUpdate', 'RoutineUpdateInput', [fullFields], `...fullFields`)
}