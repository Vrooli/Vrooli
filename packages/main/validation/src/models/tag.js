import { bool, description, opt, req, tag, transRel, yupObj } from "../utils";
export const tagTranslationValidation = transRel({
    create: {
        description: req(description),
    },
    update: {
        description: opt(description),
    },
});
export const tagValidation = {
    create: ({ o }) => yupObj({
        anonymous: opt(bool),
        tag: req(tag),
    }, [
        ["translations", ["Create"], "many", "opt", tagTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        anonymous: opt(bool),
        tag: req(tag),
    }, [
        ["translations", ["Delete", "Create", "Update"], "many", "opt", tagTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=tag.js.map