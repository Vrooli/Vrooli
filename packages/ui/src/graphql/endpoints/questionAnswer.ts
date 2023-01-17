import { questionAnswerFields as fullFields, listQuestionAnswerFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const questionAnswerEndpoint = {
    findOne: toQuery('questionAnswer', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('questionAnswers', 'QuestionAnswerSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('questionAnswerCreate', 'QuestionAnswerCreateInput', [fullFields], `...fullFields`),
    update: toMutation('questionAnswerUpdate', 'QuestionAnswerUpdateInput', [fullFields], `...fullFields`),
    accept: toMutation('questionAnswerMarkAsAccepted', 'FindByIdInput', [fullFields], `...fullFields`),
}