import { quizQuestionFields as fullFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const quizQuestionEndpoint = {
    findOne: toQuery('quizQuestion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('quizQuestions', 'QuizQuestionSearchInput', toSearch(fullFields)),
}