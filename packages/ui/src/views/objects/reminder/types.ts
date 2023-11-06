import { Reminder } from "@local/shared";
import { FormProps } from "forms/types";
import { ReminderShape } from "utils/shape/models/reminder";
import { CrudProps } from "../types";
import { NewReminderShape } from "./ReminderCrud/ReminderCrud";

export interface ReminderCrudProps extends Omit<CrudProps<Reminder>, "overrideObject"> {
    overrideObject?: NewReminderShape;
}
export interface ReminderFormProps extends FormProps<Reminder, ReminderShape> {
    index?: number;
    reminderListId?: string;
}
