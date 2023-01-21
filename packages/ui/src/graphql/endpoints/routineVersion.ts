import { routineVersionPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const routineVersionEndpoint = {
    findOne: toQuery('routineVersion', 'FindByIdInput', routineVersionPartial, 'full'),
    findMany: toQuery('routineVersions', 'RoutineVersionSearchInput', ...toSearch(routineVersionPartial)),
    create: toMutation('routineVersionCreate', 'RoutineVersionCreateInput', routineVersionPartial, 'full'),
    update: toMutation('routineVersionUpdate', 'RoutineVersionUpdateInput', routineVersionPartial, 'full')
}