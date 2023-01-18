import { statsQuizFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsQuizEndpoint = {
    findMany: toQuery('statsQuiz', 'StatsQuizSearchInput', toSearch(listFields)),
}