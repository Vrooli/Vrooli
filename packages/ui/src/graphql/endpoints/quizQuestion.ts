import { quizQuestionPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const quizQuestionEndpoint = {
    findOne: toQuery('quizQuestion', 'FindByIdInput', quizQuestionPartial, 'full'),
    findMany: toQuery('quizQuestions', 'QuizQuestionSearchInput', ...toSearch(quizQuestionPartial)),
}