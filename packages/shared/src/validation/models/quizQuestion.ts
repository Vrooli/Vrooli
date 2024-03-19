import { description, id, intPositiveOrOne, intPositiveOrZero, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { standardVersionValidation } from "./standardVersion";

export const quizQuestionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const quizQuestionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        order: opt(intPositiveOrZero),
        points: opt(intPositiveOrOne),
    }, [
        ["standardVersion", ["Connect", "Create"], "one", "opt", standardVersionValidation],
        ["quiz", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", quizQuestionTranslationValidation],
    ], [["standardVersionConnect", "standardVersionCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        order: opt(intPositiveOrZero),
        points: opt(intPositiveOrOne),
    }, [
        ["standardVersion", ["Connect", "Create", "Update"], "one", "opt", standardVersionValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", quizQuestionTranslationValidation],
    ], [], d),
};
