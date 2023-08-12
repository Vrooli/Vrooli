import { Reminder } from "@local/shared";
import { NewReminderShape } from "forms/ReminderForm/ReminderForm";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export interface ReminderUpsertProps extends Omit<UpsertProps<Reminder>, "overrideObject"> {
    handleDelete?: () => void;
    index?: number;
    overrideObject?: NewReminderShape;
}
export type ReminderViewProps = ObjectViewProps<Reminder>
