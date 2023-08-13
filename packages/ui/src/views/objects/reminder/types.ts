import { Reminder } from "@local/shared";
import { NewReminderShape } from "forms/ReminderForm/ReminderForm";
import { CrudProps } from "../types";

export interface ReminderCrudProps extends Omit<CrudProps<Reminder>, "overrideObject"> {
    overrideObject?: NewReminderShape;
}
