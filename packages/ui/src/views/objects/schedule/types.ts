import { Schedule } from "@local/shared";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { UpsertProps, ViewProps } from "../types";

export interface ScheduleUpsertProps extends UpsertProps<Schedule> {
    canChangeTab?: boolean;
    canSetScheduleFor?: boolean;
    defaultTab?: CalendarPageTabOption;
    handleDelete: () => void;
    index?: number;
    isMutate: boolean;
    listId?: string;
    partialData?: Partial<Schedule>;
}
export type ScheduleViewProps = ViewProps<Schedule>
