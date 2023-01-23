import { projectVersionPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const projectVersionEndpoint = {
    findOne: toQuery('projectVersion', 'FindByIdInput', projectVersionPartial, 'full'),
    findMany: toQuery('projectVersions', 'ProjectVersionSearchInput', ...toSearch(projectVersionPartial)),
    create: toMutation('projectVersionCreate', 'ProjectVersionCreateInput', projectVersionPartial, 'full'),
    update: toMutation('projectVersionUpdate', 'ProjectVersionUpdateInput', projectVersionPartial, 'full')
}