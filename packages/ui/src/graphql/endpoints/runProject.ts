import { runProjectFields as fullFields, listRunProjectFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const runProjectEndpoint = {
    findOne: toQuery('runProject', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('runProjects', 'RunProjectSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('runProjectCreate', 'RunProjectCreateInput', [fullFields], `...fullFields`),
    update: toMutation('runProjectUpdate', 'RunProjectUpdateInput', [fullFields], `...fullFields`),
    deleteAll: toMutation('runProjectDeleteAll', null, [], `count`),
    complete: toMutation('runProjectComplete', 'RunProjectCompleteInput', [fullFields], `...fullFields`),
    cancel: toMutation('runProjectCancel', 'RunProjectCancelInput', [fullFields], `...fullFields`),
}