import { questionFields as fullFields, listQuestionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const questionEndpoint = {
    findOne: toQuery('question', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('questions', 'QuestionSearchInput', toSearch(listFields)),
    create: toMutation('questionCreate', 'QuestionCreateInput', fullFields[1]),
    update: toMutation('questionUpdate', 'QuestionUpdateInput', fullFields[1])
}