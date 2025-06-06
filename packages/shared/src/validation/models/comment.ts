/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in comment.test.ts
import * as yup from "yup";
import { CommentFor } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr, minStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const createdFor = enumToYup(CommentFor);
const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const commentTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const commentValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        createdFor: req(createdFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["parent", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", commentTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Delete", "Create", "Update"], "many", "opt", commentTranslationValidation],
    ], [], d),
};
