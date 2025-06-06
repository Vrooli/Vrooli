/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in bookmarkList.test.ts
import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { bookmarkValidation } from "./bookmark.js";

const label = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const bookmarkListValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        label: req(label),
    }, [
        ["bookmarks", ["Connect", "Create"], "many", "opt", bookmarkValidation, ["list"]],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        label: opt(label),
    }, [
        ["bookmarks", ["Connect", "Create", "Update", "Delete"], "many", "opt", bookmarkValidation, ["list"]],
    ], [], d),
};
/* c8 ignore stop */
