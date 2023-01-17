import { id, req, opt, YupModel, rel, transRel, response } from '../utils';
import * as yup from 'yup';

export const quizQuestionResponseTranslationValidation: YupModel = transRel({
    create: {
        response: req(response),
    },
    update: {
        response: opt(response),
    }
})

export const quizQuestionResponseValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        response: opt(response),
        ...rel('quizAttempt', ['Connect'], 'one', 'opt'),
        ...rel('quizQuestion', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', quizQuestionResponseTranslationValidation),
    }, [['standardVersionConnect', 'standardVersionCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        response: opt(response),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', quizQuestionResponseTranslationValidation),
    }),
}