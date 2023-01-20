import { historyResultPartial } from 'graphql/partial/historyResult';
import { toQuery } from 'graphql/utils';

export const historyEndpoint = {
    history: toQuery('history', 'HistoryInput', historyResultPartial, 'list'),
}