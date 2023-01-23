import { standardPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const standardEndpoint = {
    findOne: toQuery('standard', 'FindByIdInput', standardPartial, 'full'),
    findMany: toQuery('standards', 'StandardSearchInput', ...toSearch(standardPartial)),
    create: toMutation('standardCreate', 'StandardCreateInput', standardPartial, 'full'),
    update: toMutation('standardUpdate', 'StandardUpdateInput', standardPartial, 'full')
}