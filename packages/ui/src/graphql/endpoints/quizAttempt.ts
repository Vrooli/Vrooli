import { quizAttemptFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizAttemptEndpoint = {
    findOne: toQuery('quizAttempt', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('quizAttempts', 'QuizAttemptSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('quizAttemptCreate', 'QuizAttemptCreateInput', [fullFields], `...fullFields`),
    update: toMutation('quizAttemptUpdate', 'QuizAttemptUpdateInput', [fullFields], `...fullFields`)
}