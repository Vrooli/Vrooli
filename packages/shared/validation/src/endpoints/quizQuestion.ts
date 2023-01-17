import { description, id, name, req, opt, YupModel, rel, transRel, intPositiveOrOne, intPositiveOrZero } from '../utils';
import * as yup from 'yup';
import { standardVersionValidation } from './standardVersion';

export const quizQuestionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const quizQuestionValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        order: opt(intPositiveOrZero),
        points: opt(intPositiveOrOne),
        ...rel('standardVersion', ['Connect', 'Create'], 'one', 'opt', standardVersionValidation),
        ...rel('quiz', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', quizQuestionTranslationValidation),
    }, [['standardVersionConnect', 'standardVersionCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        order: opt(intPositiveOrZero),
        points: opt(intPositiveOrOne),
        ...rel('standardVersion', ['Connect', 'Create', 'Update'], 'one', 'opt', standardVersionValidation),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', quizQuestionTranslationValidation),
    }),
}