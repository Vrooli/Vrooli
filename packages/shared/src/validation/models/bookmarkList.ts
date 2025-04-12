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
