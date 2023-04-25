import { id, intPositiveOrOne, language, opt, req, YupModel, yupObj } from "../utils";
import { quizQuestionResponseValidation } from "./quizQuestionResponse";

export const quizAttemptValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        timeTaken: opt(intPositiveOrOne),
        language: opt(language),
    }, [
        ["quiz", ["Connect"], "one", "req"],
        ["responses", ["Create"], "one", "opt", quizQuestionResponseValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        timeTaken: opt(intPositiveOrOne),
    }, [
        ["responses", ["Create", "Update", "Delete"], "one", "opt", quizQuestionResponseValidation],
    ], [], o),
};
