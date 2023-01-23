import { reputationHistoryPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const reputationHistoryEndpoint = {
    findOne: toQuery('reputationHistory', 'FindByIdInput', reputationHistoryPartial, 'full'),
    findMany: toQuery('reputationHistories', 'ReputationHistorySearchInput', ...toSearch(reputationHistoryPartial)),
}