import { quizQuestionPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const quizQuestionEndpoint = {
    findOne: toQuery('quizQuestion', 'FindByIdInput', quizQuestionPartial, 'full'),
    findMany: toQuery('quizQuestions', 'QuizQuestionSearchInput', ...toSearch(quizQuestionPartial)),
}