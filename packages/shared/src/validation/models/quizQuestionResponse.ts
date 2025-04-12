import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, response } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

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
