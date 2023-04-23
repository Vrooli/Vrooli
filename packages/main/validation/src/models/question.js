import { QuestionForType } from "@local/consts";
import { bool, description, enumToYup, id, name, opt, referencing, req, transRel, yupObj } from "../utils";
const forObjectType = enumToYup(QuestionForType);
export const questionTranslationValidation = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});
export const questionValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        referencing: opt(referencing),
        forObjectType: req(forObjectType),
    }, [
        ["forObject", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", questionTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["acceptedAnswer", ["Connect"], "one", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", questionTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=question.js.map