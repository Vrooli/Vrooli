import { Reminder } from ":local/consts";
import { UpsertProps, ViewProps } from "../types";

export interface ReminderUpsertProps extends UpsertProps<Reminder> {
    handleDelete: () => void;
    index?: number;
    listId?: string;
    partialData?: Partial<Reminder>;
    reminderListId?: string;
}
export interface ReminderViewProps extends ViewProps<Reminder> { }
