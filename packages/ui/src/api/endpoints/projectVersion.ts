import { projectVersionPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const projectVersionEndpoint = {
    findOne: toQuery('projectVersion', 'FindByIdInput', projectVersionPartial, 'full'),
    findMany: toQuery('projectVersions', 'ProjectVersionSearchInput', ...toSearch(projectVersionPartial)),
    create: toMutation('projectVersionCreate', 'ProjectVersionCreateInput', projectVersionPartial, 'full'),
    update: toMutation('projectVersionUpdate', 'ProjectVersionUpdateInput', projectVersionPartial, 'full')
}