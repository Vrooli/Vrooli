import { Reminder, ReminderItem, ReminderList } from "@shared/consts";
import { ReminderShape } from "utils/shape/models/reminder";
import { ReminderItemShape } from "utils/shape/models/reminderItem";

export interface ReminderListProps {
    handleUpdate?: (updatedList: ReminderList) => void;
    list: ReminderList | null | undefined;
    loading: boolean;
    zIndex: number;
}

export interface ReminderProps {
    reminder: Reminder | ReminderShape;
    handleDelete: (deletedReminder: Reminder) => void;
    handleUpdate: (updatedReminder: Reminder) => void;
    zIndex: number;
}

export interface ReminderItemProps {
    reminderItem: ReminderItem | ReminderItemShape;
    handleDelete: (deletedReminderItem: ReminderItem) => void;
    handleUpdate: (updatedReminderItem: ReminderItem) => void;
}