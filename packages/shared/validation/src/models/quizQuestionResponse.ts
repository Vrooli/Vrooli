import { id, req, opt, YupModel, transRel, response, yupObj } from '../utils';

export const quizQuestionResponseTranslationValidation: YupModel = transRel({
    create: {
        response: req(response),
    },
    update: {
        response: opt(response),
    }
})

export const quizQuestionResponseValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        response: req(response),
    }, [
        ['quizAttempt', ['Connect'], 'one', 'opt'],
        ['quizQuestion', ['Connect'], 'one', 'opt'],
    ], [['standardVersionConnect', 'standardVersionCreate']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        response: opt(response),
    }, [], [], o),
}