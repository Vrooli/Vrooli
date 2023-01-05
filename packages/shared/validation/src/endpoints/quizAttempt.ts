import { id, req, opt, YupModel, rel, transRel, response, intPositiveOrOne, language } from '../utils';
import * as yup from 'yup';
import { quizQuestionResponseValidation } from './quizQuestionResponse';

export const quizAttemptValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        timeTaken: opt(intPositiveOrOne),
        language: opt(language),
        ...rel('responses', ['Create'], 'one', 'opt', quizQuestionResponseValidation),
    }, [['standardVersionConnect', 'standardVersionCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        timeTaken: opt(intPositiveOrOne),
        ...rel('responses', ['Create', 'Update', 'Delete'], 'one', 'opt', quizQuestionResponseValidation),
    }),
}