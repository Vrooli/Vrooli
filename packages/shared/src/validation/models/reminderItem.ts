/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in reminderItem.test.ts
/* c8 ignore end */

import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id } from "../utils/commonFields.js";
import { maxStrErr, minNumErr, minStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const REMINDER_ITEM_NAME_MAX_LENGTH = 50;

const index = yup.number().integer().min(0, minNumErr);
const name = yup.string().trim().removeEmptyString().min(1, minStrErr).max(REMINDER_ITEM_NAME_MAX_LENGTH, maxStrErr);

export const reminderItemValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
        isComplete: opt(bool),
        name: req(name),
    }, [
        ["reminder", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
        isComplete: opt(bool),
        name: opt(name),
    }, [], [], d),
};
