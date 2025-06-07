/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in scheduleRecurrence.test.ts
import * as yup from "yup";
import { ScheduleRecurrenceType } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { endDate, id, intPositiveOrOne } from "../utils/commonFields.js";
import { maxNumErr, minNumErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { DAY_OF_MONTH_MAX, DAY_OF_MONTH_MIN, DAY_OF_WEEK_MAX, DAY_OF_WEEK_MIN } from "../utils/validationConstants.js";
import { scheduleValidation } from "./schedule.js";

function recurrenceType() {
    return enumToYup(ScheduleRecurrenceType);
}
const dayOfWeek = yup.number().min(DAY_OF_WEEK_MIN, minNumErr).max(DAY_OF_WEEK_MAX, maxNumErr).integer();
const dayOfMonth = yup.number().min(DAY_OF_MONTH_MIN, minNumErr).max(DAY_OF_MONTH_MAX, maxNumErr).integer();

export const scheduleRecurrenceValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        recurrenceType: req(recurrenceType()),
        interval: req(intPositiveOrOne),
        dayOfWeek: opt(dayOfWeek),
        dayOfMonth: opt(dayOfMonth),
        duration: opt(intPositiveOrOne),
        endDate: opt(endDate),
    }, [
        ["schedule", ["Connect"], "one", "req", scheduleValidation, ["recurrences"]],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        recurrenceType: opt(recurrenceType()),
        interval: opt(intPositiveOrOne),
        dayOfWeek: opt(dayOfWeek),
        dayOfMonth: opt(dayOfMonth),
        duration: opt(intPositiveOrOne),
        endDate: opt(endDate),
    }, [], [], d),
};
/* c8 ignore stop */
