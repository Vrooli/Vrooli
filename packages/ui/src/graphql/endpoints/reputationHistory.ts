import { reputationHistoryFields as fullFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const reputationHistoryEndpoint = {
    findOne: toQuery('reputationHistory', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('reputationHistories', 'ReputationHistorySearchInput', toSearch(fullFields)),
}