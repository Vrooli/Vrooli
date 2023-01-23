import { quizQuestionPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const quizQuestionEndpoint = {
    findOne: toQuery('quizQuestion', 'FindByIdInput', quizQuestionPartial, 'full'),
    findMany: toQuery('quizQuestions', 'QuizQuestionSearchInput', ...toSearch(quizQuestionPartial)),
}