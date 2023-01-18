import { quizAttemptFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizAttemptEndpoint = {
    findOne: toQuery('quizAttempt', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('quizAttempts', 'QuizAttemptSearchInput', toSearch(fullFields)),
    create: toMutation('quizAttemptCreate', 'QuizAttemptCreateInput', fullFields[1]),
    update: toMutation('quizAttemptUpdate', 'QuizAttemptUpdateInput', fullFields[1])
}