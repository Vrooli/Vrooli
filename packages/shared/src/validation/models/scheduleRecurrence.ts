import * as yup from "yup";
import { ScheduleRecurrenceType } from "../../api/types";
import { endDate, enumToYup, id, intPositiveOrOne, maxNumErr, minNumErr, opt, req, YupModel, yupObj } from "../utils";

const recurrenceType = enumToYup(ScheduleRecurrenceType);
const dayOfWeek = yup.number().min(1, minNumErr).max(7, maxNumErr).integer();
const dayOfMonth = yup.number().min(1, minNumErr).max(31, maxNumErr).integer();

export const scheduleRecurrenceValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        recurrenceType: req(recurrenceType),
        interval: req(intPositiveOrOne),
        dayOfWeek: opt(dayOfWeek),
        dayOfMonth: opt(dayOfMonth),
        endDate: opt(endDate),
    }, [
        ["schedule", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        recurrenceType: opt(recurrenceType),
        interval: opt(intPositiveOrOne),
        dayOfWeek: opt(dayOfWeek),
        dayOfMonth: opt(dayOfMonth),
        endDate: opt(endDate),
    }, [], [], d),
};
