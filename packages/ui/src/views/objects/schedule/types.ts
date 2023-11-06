import { Schedule } from "@local/shared";
import { FormProps } from "forms/types";
import { PageTab } from "hooks/useTabs";
import { ChangeEvent } from "react";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { ScheduleShape } from "utils/shape/models/schedule";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export interface ScheduleUpsertProps extends UpsertProps<Schedule> {
    canChangeTab?: boolean;
    canSetScheduleFor?: boolean;
    defaultTab?: CalendarPageTabOption;
    handleDelete: () => unknown;
    index?: number;
    isMutate: boolean;
    listId?: string;
}
export type ScheduleFormProps = FormProps<Schedule, ScheduleShape> & Pick<ScheduleUpsertProps, "canChangeTab" | "canSetScheduleFor" | "isMutate"> & {
    currTab: PageTab<CalendarPageTabOption, true>;
    handleTabChange: (e: ChangeEvent<unknown> | undefined, tab: PageTab<CalendarPageTabOption, true>) => unknown;
    tabs: PageTab<CalendarPageTabOption, true>[];
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
