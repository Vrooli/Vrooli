import { Reminder } from "@local/shared";
import { CrudProps } from "../types";
import { NewReminderShape } from "./ReminderCrud/ReminderCrud";

export interface ReminderCrudProps extends Omit<CrudProps<Reminder>, "overrideObject"> {
    overrideObject?: NewReminderShape;
}
