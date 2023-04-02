import { Reminder } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface ReminderUpsertProps extends UpsertProps<Reminder> {
    index?: number;
    reminderListId?: string;
}
export interface ReminderViewProps extends ViewProps<Reminder> { }