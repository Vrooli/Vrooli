import { Reminder, ReminderShape } from "@local/shared";
import { CrudProps, FormProps } from "../../../types.js";

export type ReminderCrudProps = CrudProps<Reminder>
export interface ReminderFormProps extends FormProps<Reminder, ReminderShape> {
    index?: number;
    reminderListId?: string;
}
