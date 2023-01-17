import { quizQuestionResponseFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizQuestionResponseEndpoint = {
    findOne: toQuery('quizQuestionResponse', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('quizQuestionResponses', 'QuizQuestionResponseSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('quizQuestionResponseCreate', 'QuizQuestionResponseCreateInput', [fullFields], `...fullFields`),
    update: toMutation('quizQuestionResponseUpdate', 'QuizQuestionResponseUpdateInput', [fullFields], `...fullFields`)
}