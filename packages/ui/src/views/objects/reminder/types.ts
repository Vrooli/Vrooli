import { Reminder } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface ReminderCreateProps extends CreateProps<Reminder> {}
export interface ReminderUpdateProps extends UpdateProps<Reminder> {}
export interface ReminderViewProps extends ViewProps<Reminder> {}