import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, tag } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

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
