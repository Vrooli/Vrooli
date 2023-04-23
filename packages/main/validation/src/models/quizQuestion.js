import { description, id, name, req, opt, transRel, intPositiveOrOne, intPositiveOrZero, yupObj } from "../utils";
import { standardVersionValidation } from "./standardVersion";
export const quizQuestionTranslationValidation = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});
export const quizQuestionValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        order: opt(intPositiveOrZero),
        points: opt(intPositiveOrOne),
    }, [
        ["standardVersion", ["Connect", "Create"], "one", "opt", standardVersionValidation],
        ["quiz", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", quizQuestionTranslationValidation],
    ], [["standardVersionConnect", "standardVersionCreate"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        order: opt(intPositiveOrZero),
        points: opt(intPositiveOrOne),
    }, [
        ["standardVersion", ["Connect", "Create", "Update"], "one", "opt", standardVersionValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", quizQuestionTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=quizQuestion.js.map