import * as yup from "yup";
import { bool, id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";

const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const chatMessageTranslationValidation: YupModel = transRel({
    create: {
        text: req(text),
    },
    update: {
        text: opt(text),
    },
});

export const chatMessageValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        isFork: opt(bool),
        forkId: opt(id),
    }, [
        ["chat", ["Connect"], "one", "req", ["messages"]],
        ["translations", ["Create"], "many", "opt", chatMessageTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", chatMessageTranslationValidation],
    ], [], o),
};
