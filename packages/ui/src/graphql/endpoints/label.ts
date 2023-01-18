import { labelFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const labelEndpoint = {
    findOne: toQuery('label', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('labels', 'LabelSearchInput', toSearch(fullFields)),
    create: toMutation('labelCreate', 'LabelCreateInput', fullFields[1]),
    update: toMutation('labelUpdate', 'LabelUpdateInput', fullFields[1])
}