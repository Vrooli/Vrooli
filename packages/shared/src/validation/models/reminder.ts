/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in reminder.test.ts
/* c8 ignore end */

import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, id, index, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { reminderItemValidation } from "./reminderItem.js";
import { reminderListValidation } from "./reminderList.js";

export const reminderValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        name: req(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ["reminderList", ["Connect", "Create"], "one", "opt", reminderListValidation, ["reminders"]],
        ["reminderItems", ["Create"], "many", "opt", reminderItemValidation],
    ], [["reminderListConnect", "reminderListCreate", false]], d),
    update: (d) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ["reminderList", ["Connect", "Create"], "one", "opt", reminderListValidation, ["reminders"]],
        ["reminderItems", ["Create", "Update", "Delete"], "many", "opt", reminderItemValidation],
    ], [], d),
};
