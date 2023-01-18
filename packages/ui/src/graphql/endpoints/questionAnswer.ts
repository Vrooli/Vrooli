import { questionAnswerFields as fullFields, listQuestionAnswerFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const questionAnswerEndpoint = {
    findOne: toQuery('questionAnswer', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('questionAnswers', 'QuestionAnswerSearchInput', toSearch(listFields)),
    create: toMutation('questionAnswerCreate', 'QuestionAnswerCreateInput', fullFields[1]),
    update: toMutation('questionAnswerUpdate', 'QuestionAnswerUpdateInput', fullFields[1]),
    accept: toMutation('questionAnswerMarkAsAccepted', 'FindByIdInput', fullFields[1]),
}