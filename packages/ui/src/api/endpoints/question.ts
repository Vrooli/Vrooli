import { questionPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const questionEndpoint = {
    findOne: toQuery('question', 'FindByIdInput', questionPartial, 'full'),
    findMany: toQuery('questions', 'QuestionSearchInput', ...toSearch(questionPartial)),
    create: toMutation('questionCreate', 'QuestionCreateInput', questionPartial, 'full'),
    update: toMutation('questionUpdate', 'QuestionUpdateInput', questionPartial, 'full')
}