import { Reminder } from "@local/shared";

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
