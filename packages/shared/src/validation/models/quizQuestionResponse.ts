import { id, opt, req, response, transRel, YupModel, yupObj } from "../utils";

export const quizQuestionResponseTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        response: req(response),
    }),
    update: () => ({
        response: opt(response),
    }),
});

export const quizQuestionResponseValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        response: req(response),
    }, [
        ["quizAttempt", ["Connect"], "one", "opt"],
        ["quizQuestion", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        response: opt(response),
    }, [], [], d),
};
