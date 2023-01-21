import { questionAnswerPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const questionAnswerEndpoint = {
    findOne: toQuery('questionAnswer', 'FindByIdInput', questionAnswerPartial, 'full'),
    findMany: toQuery('questionAnswers', 'QuestionAnswerSearchInput', ...toSearch(questionAnswerPartial)),
    create: toMutation('questionAnswerCreate', 'QuestionAnswerCreateInput', questionAnswerPartial, 'full'),
    update: toMutation('questionAnswerUpdate', 'QuestionAnswerUpdateInput', questionAnswerPartial, 'full'),
    accept: toMutation('questionAnswerMarkAsAccepted', 'FindByIdInput', questionAnswerPartial, 'full'),
}