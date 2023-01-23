import { quizPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const quizEndpoint = {
    findOne: toQuery('quiz', 'FindByIdInput', quizPartial, 'full'),
    findMany: toQuery('quizzes', 'QuizSearchInput', ...toSearch(quizPartial)),
    create: toMutation('quizCreate', 'QuizCreateInput', quizPartial, 'full'),
    update: toMutation('quizUpdate', 'QuizUpdateInput', quizPartial, 'full')
}