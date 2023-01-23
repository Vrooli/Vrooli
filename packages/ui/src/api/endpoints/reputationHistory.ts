import { reputationHistoryPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const reputationHistoryEndpoint = {
    findOne: toQuery('reputationHistory', 'FindByIdInput', reputationHistoryPartial, 'full'),
    findMany: toQuery('reputationHistories', 'ReputationHistorySearchInput', ...toSearch(reputationHistoryPartial)),
}