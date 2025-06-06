/* c8 ignore start */
// Coverage is ignored for validation model files because they export schema objects rather than
// executable functions. Their correctness is verified through comprehensive validation tests.
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { chatInviteValidation } from "./chatInvite.js";
import { chatMessageValidation } from "./chatMessage.js";

export const chatTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        name: opt(name),
        description: opt(description),
    }),
    update: () => ({
        name: opt(name),
        description: opt(description),
    }),
});

export const chatValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
    }, [
        ["invites", ["Create"], "many", "opt", chatInviteValidation, ["chat"]],
        ["messages", ["Create"], "many", "opt", chatMessageValidation, ["chat"]],
        ["team", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", chatTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
    }, [
        ["invites", ["Create", "Update", "Delete"], "many", "opt", chatInviteValidation],
        ["messages", ["Create", "Update", "Delete"], "many", "opt", chatMessageValidation],
        ["participants", ["Delete"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", chatTranslationValidation],
    ], [], d),
};
