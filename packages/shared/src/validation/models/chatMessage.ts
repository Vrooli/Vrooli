import * as yup from "yup";
import { id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";
import { chatValidation } from "./chat";

const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const chatMessageTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const chatMessageValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        versionOfId: opt(id),
    }, [
        ["chat", ["Connect"], "one", "req", chatValidation, ["messages"]],
        ["user", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", chatMessageTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", chatMessageTranslationValidation],
    ], [], d),
};
