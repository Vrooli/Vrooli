import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, id, name, permissions } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const roleTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: req(description),
    }),
    update: () => ({
        description: opt(description),
    }),
});

export const roleValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
    }, [
        ["members", ["Connect"], "many", "opt"],
        ["team", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", roleTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
    }, [
        ["members", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", roleTranslationValidation],
    ], [], d),
};
