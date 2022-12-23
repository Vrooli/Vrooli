import { questionFields as fullFields, listQuestionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const questionEndpoint = {
    findOne: toQuery('question', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('questions', 'QuestionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('questionCreate', 'QuestionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('questionUpdate', 'QuestionUpdateInput', [fullFields], `...fullFields`)
}