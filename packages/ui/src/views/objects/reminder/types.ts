import { Reminder } from "@local/shared";
import { UpsertProps, ViewProps } from "../types";

export interface ReminderUpsertProps extends UpsertProps<Reminder> {
    handleDelete?: () => void;
    index?: number;
    listId?: string;
    partialData?: Partial<Reminder>;
    reminderListId?: string;
}
export type ReminderViewProps = ViewProps<Reminder>
