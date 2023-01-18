import { quizQuestionResponseFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizQuestionResponseEndpoint = {
    findOne: toQuery('quizQuestionResponse', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('quizQuestionResponses', 'QuizQuestionResponseSearchInput', toSearch(fullFields)),
    create: toMutation('quizQuestionResponseCreate', 'QuizQuestionResponseCreateInput', fullFields[1]),
    update: toMutation('quizQuestionResponseUpdate', 'QuizQuestionResponseUpdateInput', fullFields[1])
}