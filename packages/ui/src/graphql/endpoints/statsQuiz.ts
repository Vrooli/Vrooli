import { statsQuizPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsQuizEndpoint = {
    findMany: toQuery('statsQuiz', 'StatsQuizSearchInput', ...toSearch(statsQuizPartial)),
}