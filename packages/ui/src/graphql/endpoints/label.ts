import { labelFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const labelEndpoint = {
    findOne: toQuery('label', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('labels', 'LabelSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('labelCreate', 'LabelCreateInput', [fullFields], `...fullFields`),
    update: toMutation('labelUpdate', 'LabelUpdateInput', [fullFields], `...fullFields`)
}