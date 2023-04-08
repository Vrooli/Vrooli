import { Reminder, ReminderItem } from "@shared/consts";

export interface ReminderListProps {
    handleUpdate?: (updatedList: Reminder[]) => void;
    /**
     * If no listId is provided, make sure list items have data 
     * about the list they belong to.
     */
    listId?: string | null | undefined;
    loading: boolean;
    reminders: Reminder[];
    zIndex: number;
}

export interface ReminderListItemProps {
    handleDelete: (deletedReminder: Reminder) => void;
    handleUpdate: (updatedReminder: Reminder) => void;
    reminder: Reminder;
    zIndex: number;
}

export interface ReminderListSubItemProps {
    reminderItem: ReminderItem;
    handleDelete: (deletedReminderItem: ReminderItem) => void;
    handleUpdate: (updatedReminderItem: ReminderItem) => void;
}