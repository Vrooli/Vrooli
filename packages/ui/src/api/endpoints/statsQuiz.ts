import { statsQuizPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsQuizEndpoint = {
    findMany: toQuery('statsQuiz', 'StatsQuizSearchInput', ...toSearch(statsQuizPartial)),
}