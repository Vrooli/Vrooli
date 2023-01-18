import { quizFields as fullFields, listQuizFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizEndpoint = {
    findOne: toQuery('quiz', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('quizzes', 'QuizSearchInput', toSearch(listFields)),
    create: toMutation('quizCreate', 'QuizCreateInput', fullFields[1]),
    update: toMutation('quizUpdate', 'QuizUpdateInput', fullFields[1])
}