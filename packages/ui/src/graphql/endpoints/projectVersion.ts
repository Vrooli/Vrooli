import { projectVersionFields as fullFields, listProjectVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const projectVersionEndpoint = {
    findOne: toQuery('projectVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('projectVersions', 'ProjectVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('projectVersionCreate', 'ProjectVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('projectVersionUpdate', 'ProjectVersionUpdateInput', [fullFields], `...fullFields`)
}