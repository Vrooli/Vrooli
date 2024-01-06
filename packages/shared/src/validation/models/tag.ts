import { bool, description, opt, req, tag, transRel, YupModel, yupObj } from "../utils";

export const tagTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: req(description),
    }),
    update: () => ({
        description: opt(description),
    }),
});

export const tagValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        anonymous: opt(bool),
        tag: req(tag),
    }, [
        ["translations", ["Create"], "many", "opt", tagTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        anonymous: opt(bool),
        tag: req(tag),
    }, [
        ["translations", ["Delete", "Create", "Update"], "many", "opt", tagTranslationValidation],
    ], [], d),
};
