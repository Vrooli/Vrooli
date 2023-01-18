import { projectVersionFields as fullFields, listProjectVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const projectVersionEndpoint = {
    findOne: toQuery('projectVersion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('projectVersions', 'ProjectVersionSearchInput', toSearch(listFields)),
    create: toMutation('projectVersionCreate', 'ProjectVersionCreateInput', fullFields[1]),
    update: toMutation('projectVersionUpdate', 'ProjectVersionUpdateInput', fullFields[1])
}