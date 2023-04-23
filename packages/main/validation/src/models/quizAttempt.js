import { id, req, opt, intPositiveOrOne, language, yupObj } from "../utils";
import { quizQuestionResponseValidation } from "./quizQuestionResponse";
export const quizAttemptValidation = {
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
//# sourceMappingURL=quizAttempt.js.map