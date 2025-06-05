import { type Reminder, type ReminderShape } from "@vrooli/shared";
import { type CrudProps, type FormProps } from "../../../types.js";

export type ReminderCrudProps = CrudProps<Reminder>
export interface ReminderFormProps extends FormProps<Reminder, ReminderShape> {
    index?: number;
    reminderListId?: string;
}
