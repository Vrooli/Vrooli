import { projectFields as fullFields, listProjectFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const projectEndpoint = {
    findOne: toQuery('project', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('projects', 'ProjectSearchInput', toSearch(listFields)),
    create: toMutation('projectCreate', 'ProjectCreateInput', fullFields[1]),
    update: toMutation('projectUpdate', 'ProjectUpdateInput', fullFields[1])
}