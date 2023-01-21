import { quizQuestionResponsePartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const quizQuestionResponseEndpoint = {
    findOne: toQuery('quizQuestionResponse', 'FindByIdInput', quizQuestionResponsePartial, 'full'),
    findMany: toQuery('quizQuestionResponses', 'QuizQuestionResponseSearchInput', ...toSearch(quizQuestionResponsePartial)),
    create: toMutation('quizQuestionResponseCreate', 'QuizQuestionResponseCreateInput', quizQuestionResponsePartial, 'full'),
    update: toMutation('quizQuestionResponseUpdate', 'QuizQuestionResponseUpdateInput', quizQuestionResponsePartial, 'full')
}