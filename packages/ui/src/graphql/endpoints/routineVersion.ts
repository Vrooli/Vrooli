import { routineVersionFields as fullFields, listRoutineVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const routineVersionEndpoint = {
    findOne: toQuery('routineVersion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('routineVersions', 'RoutineVersionSearchInput', toSearch(listFields)),
    create: toMutation('routineVersionCreate', 'RoutineVersionCreateInput', fullFields[1]),
    update: toMutation('routineVersionUpdate', 'RoutineVersionUpdateInput', fullFields[1])
}