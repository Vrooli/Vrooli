import { Reminder, ReminderShape } from "@local/shared";
import { FormProps } from "forms/types";
import { CrudProps } from "../types";

export type ReminderCrudProps = CrudProps<Reminder>
export interface ReminderFormProps extends FormProps<Reminder, ReminderShape> {
    index?: number;
    reminderListId?: string;
}
