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
