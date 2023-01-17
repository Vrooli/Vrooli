import { projectFields as fullFields, listProjectFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const projectEndpoint = {
    findOne: toQuery('project', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('projects', 'ProjectSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('projectCreate', 'ProjectCreateInput', [fullFields], `...fullFields`),
    update: toMutation('projectUpdate', 'ProjectUpdateInput', [fullFields], `...fullFields`)
}