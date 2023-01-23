import { statsQuizPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsQuizEndpoint = {
    findMany: toQuery('statsQuiz', 'StatsQuizSearchInput', ...toSearch(statsQuizPartial)),
}