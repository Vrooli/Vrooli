import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, hexColor, id } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const label = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const labelTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: req(description),
    }),
    update: () => ({
        description: opt(description),
    }),
});

export const labelValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        label: req(label),
        color: opt(hexColor),
    }, [
        ["apis", ["Connect"], "many", "opt"],
        ["codes", ["Connect"], "many", "opt"],
        ["issues", ["Connect"], "many", "opt"],
        ["notes", ["Connect"], "many", "opt"],
        ["projects", ["Connect"], "many", "opt"],
        ["routines", ["Connect"], "many", "opt"],
        ["standards", ["Connect"], "many", "opt"],
        ["meetings", ["Connect"], "many", "opt"],
        ["schedules", ["Connect"], "many", "opt"],
        ["team", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", labelTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        label: req(label),
        color: opt(hexColor),
    }, [
        ["apis", ["Connect", "Disconnect"], "many", "opt"],
        ["codes", ["Connect", "Disconnect"], "many", "opt"],
        ["issues", ["Connect", "Disconnect"], "many", "opt"],
        ["notes", ["Connect", "Disconnect"], "many", "opt"],
        ["projects", ["Connect", "Disconnect"], "many", "opt"],
        ["routines", ["Connect", "Disconnect"], "many", "opt"],
        ["standards", ["Connect", "Disconnect"], "many", "opt"],
        ["meetings", ["Connect", "Disconnect"], "many", "opt"],
        ["schedules", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", labelTranslationValidation],
    ], [], d),
};
