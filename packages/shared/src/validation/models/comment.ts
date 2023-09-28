import { CommentFor } from "@local/shared";
import * as yup from "yup";
import { enumToYup, id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";

const createdFor = enumToYup(CommentFor);
const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const commentTranslationValidation: YupModel = transRel({
    create: {
        text: req(text),
    },
    update: {
        text: opt(text),
    },
});

export const commentValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        createdFor: req(createdFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["parent", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", commentTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Delete", "Create", "Update"], "many", "opt", commentTranslationValidation],
    ], [], o),
};
