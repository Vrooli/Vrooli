import { Reminder } from "@local/shared";
import { FormProps } from "forms/types";
import { ReminderShape } from "utils/shape/models/reminder";
import { CrudProps } from "../types";

export type ReminderCrudProps = CrudProps<Reminder>
export interface ReminderFormProps extends FormProps<Reminder, ReminderShape> {
    index?: number;
    reminderListId?: string;
}
