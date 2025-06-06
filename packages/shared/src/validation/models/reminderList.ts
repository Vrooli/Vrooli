/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in reminderList.test.ts
/* c8 ignore end */

import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { reminderValidation } from "./reminder.js";

export const reminderListValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
    }, [
        ["reminders", ["Create"], "many", "opt", reminderValidation, ["reminderList"]],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["reminders", ["Create", "Update", "Delete"], "many", "opt", reminderValidation, ["reminderList"]],
    ], [], d),
};
