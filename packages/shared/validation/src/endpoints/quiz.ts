import { description, id, name, req, opt, YupModel, rel, transRel, intPositiveOrOne } from '../utils';
import * as yup from 'yup';
import { quizQuestionValidation } from './quizQuestion';

export const quizTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const quizValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        maxAttempts: opt(intPositiveOrOne),
        randomizeQuestionOrder: opt(yup.boolean()),
        revealCorrectAnswers: opt(yup.boolean()),
        timeLimit: opt(intPositiveOrOne),
        pointsToPass: opt(intPositiveOrOne),
        ...rel('routine', ['Connect'], 'one', 'opt'),
        ...rel('project', ['Connect'], 'one', 'opt'),
        ...rel('quizQuestions', ['Create'], 'many', 'opt', quizQuestionValidation),
        ...rel('translations', ['Create'], 'many', 'opt', quizTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        maxAttempts: opt(intPositiveOrOne),
        randomizeQuestionOrder: opt(yup.boolean()),
        revealCorrectAnswers: opt(yup.boolean()),
        timeLimit: opt(intPositiveOrOne),
        pointsToPass: opt(intPositiveOrOne),
        ...rel('routine', ['Connect', 'Disconnect'], 'one', 'opt'),
        ...rel('project', ['Connect', 'Disconnect'], 'one', 'opt'),
        ...rel('quizQuestions', ['Create', 'Update', 'Delete'], 'many', 'opt', quizQuestionValidation),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', quizTranslationValidation),
    }),
}