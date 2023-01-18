import { runProjectFields as fullFields, listRunProjectFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runProjectEndpoint = {
    findOne: toQuery('runProject', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('runProjects', 'RunProjectSearchInput', toSearch(listFields)),
    create: toMutation('runProjectCreate', 'RunProjectCreateInput', fullFields[1]),
    update: toMutation('runProjectUpdate', 'RunProjectUpdateInput', fullFields[1]),
    deleteAll: toMutation('runProjectDeleteAll', null, `{ count }`),
    complete: toMutation('runProjectComplete', 'RunProjectCompleteInput', fullFields[1]),
    cancel: toMutation('runProjectCancel', 'RunProjectCancelInput', fullFields[1]),
}