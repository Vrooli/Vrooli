import { countPartial, runProjectPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runProjectEndpoint = {
    findOne: toQuery('runProject', 'FindByIdInput', runProjectPartial, 'full'),
    findMany: toQuery('runProjects', 'RunProjectSearchInput', ...toSearch(runProjectPartial)),
    create: toMutation('runProjectCreate', 'RunProjectCreateInput', runProjectPartial, 'full'),
    update: toMutation('runProjectUpdate', 'RunProjectUpdateInput', runProjectPartial, 'full'),
    deleteAll: toMutation('runProjectDeleteAll', null, countPartial, 'full'),
    complete: toMutation('runProjectComplete', 'RunProjectCompleteInput', runProjectPartial, 'full'),
    cancel: toMutation('runProjectCancel', 'RunProjectCancelInput', runProjectPartial, 'full'),
}