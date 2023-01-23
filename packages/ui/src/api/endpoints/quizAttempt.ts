import { quizAttemptPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const quizAttemptEndpoint = {
    findOne: toQuery('quizAttempt', 'FindByIdInput', quizAttemptPartial, 'full'),
    findMany: toQuery('quizAttempts', 'QuizAttemptSearchInput', ...toSearch(quizAttemptPartial)),
    create: toMutation('quizAttemptCreate', 'QuizAttemptCreateInput', quizAttemptPartial, 'full'),
    update: toMutation('quizAttemptUpdate', 'QuizAttemptUpdateInput', quizAttemptPartial, 'full')
}