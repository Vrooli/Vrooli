import { quizQuestionFields as fullFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const quizQuestionEndpoint = {
    findOne: toQuery('quizQuestion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('quizQuestions', 'QuizQuestionSearchInput', [fullFields], toSearch(fullFields)),
}