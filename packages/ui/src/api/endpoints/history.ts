import { historyResultPartial } from 'api/partial/historyResult';
import { toQuery } from 'api/utils';

export const historyEndpoint = {
    history: toQuery('history', 'HistoryInput', historyResultPartial, 'list'),
}