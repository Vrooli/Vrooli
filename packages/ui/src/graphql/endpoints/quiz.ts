import { quizFields as fullFields, listQuizFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizEndpoint = {
    findOne: toQuery('quiz', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('quizzes', 'QuizSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('quizCreate', 'QuizCreateInput', [fullFields], `...fullFields`),
    update: toMutation('quizUpdate', 'QuizUpdateInput', [fullFields], `...fullFields`)
}