import { Reminder } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export interface ReminderUpsertProps extends UpsertProps<Reminder> {
    handleDelete?: () => void;
    index?: number;
    listId?: string;
    reminderListId?: string;
}
export type ReminderViewProps = ObjectViewProps<Reminder>
