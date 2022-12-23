import { reputationHistoryFields as fullFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const reputationHistoryEndpoint = {
    findOne: toQuery('reputationHistory', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('reputationHistories', 'ReputationHistorySearchInput', [fullFields], toSearch(fullFields)),
}