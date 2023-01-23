import { labelPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const labelEndpoint = {
    findOne: toQuery('label', 'FindByIdInput', labelPartial, 'full'),
    findMany: toQuery('labels', 'LabelSearchInput', ...toSearch(labelPartial)),
    create: toMutation('labelCreate', 'LabelCreateInput', labelPartial, 'full'),
    update: toMutation('labelUpdate', 'LabelUpdateInput', labelPartial, 'full')
}