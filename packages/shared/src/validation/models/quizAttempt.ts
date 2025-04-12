import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, intPositiveOrOne, language } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { quizQuestionResponseValidation } from "./quizQuestionResponse.js";

export const quizAttemptValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        timeTaken: opt(intPositiveOrOne),
        language: opt(language),
    }, [
        ["quiz", ["Connect"], "one", "req"],
        ["responses", ["Create"], "one", "opt", quizQuestionResponseValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        timeTaken: opt(intPositiveOrOne),
    }, [
        ["responses", ["Create", "Update", "Delete"], "one", "opt", quizQuestionResponseValidation],
    ], [], d),
};
