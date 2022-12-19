import { description, id, name, req, opt, rel, YupModel, transRel, index } from '../utils';
import * as yup from 'yup';
import { routineVersionValidation } from './routineVersion';

const isOptional = yup.boolean()

export const nodeRoutineListItemTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
})

export const nodeRoutineListItemValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        index: req(index),
        isOptional: opt(isOptional),
        ...rel('list', ['Connect'], 'one', 'req'),
        ...rel('routineVersion', ['Connect'], 'one', 'req'), // Creating subroutines must be done in a separate request
        ...rel('translations', ['Create'], 'many', 'opt', nodeRoutineListItemTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        index: opt(index),
        isOptional: opt(isOptional),
        ...rel('routineVersion', ['Update'], 'one', 'opt', routineVersionValidation), // Create/update/delete of subroutines must be done in a separate request
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeRoutineListItemTranslationValidation),
    }),
}