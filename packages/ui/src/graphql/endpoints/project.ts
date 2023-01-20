import { projectPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const projectEndpoint = {
    findOne: toQuery('project', 'FindByIdInput', projectPartial, 'full'),
    findMany: toQuery('projects', 'ProjectSearchInput', ...toSearch(projectPartial)),
    create: toMutation('projectCreate', 'ProjectCreateInput', projectPartial, 'full'),
    update: toMutation('projectUpdate', 'ProjectUpdateInput', projectPartial, 'full')
}