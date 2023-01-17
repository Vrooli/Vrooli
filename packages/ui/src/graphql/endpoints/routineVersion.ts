import { routineVersionFields as fullFields, listRoutineVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const routineVersionEndpoint = {
    findOne: toQuery('routineVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('routineVersions', 'RoutineVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('routineVersionCreate', 'RoutineVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('routineVersionUpdate', 'RoutineVersionUpdateInput', [fullFields], `...fullFields`)
}