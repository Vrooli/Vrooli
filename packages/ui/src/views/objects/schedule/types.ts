import { Schedule } from "@local/shared";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export interface ScheduleUpsertProps extends UpsertProps<Schedule> {
    canChangeTab?: boolean;
    canSetScheduleFor?: boolean;
    defaultTab?: CalendarPageTabOption;
    handleDelete: () => void;
    index?: number;
    isMutate: boolean;
    listId?: string;
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
