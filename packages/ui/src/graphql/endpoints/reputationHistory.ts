import { reputationHistoryPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const reputationHistoryEndpoint = {
    findOne: toQuery('reputationHistory', 'FindByIdInput', reputationHistoryPartial, 'full'),
    findMany: toQuery('reputationHistories', 'ReputationHistorySearchInput', ...toSearch(reputationHistoryPartial)),
}