import { historyResultPartial } from '../partial/historyResult';
import { toQuery } from '../utils';

export const historyEndpoint = {
    history: toQuery('history', 'HistoryInput', historyResultPartial, 'list'),
}