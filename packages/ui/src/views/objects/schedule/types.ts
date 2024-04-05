import { Schedule } from "@local/shared";
import { FormProps } from "forms/types";
import { PageTab } from "hooks/useTabs";
import { ChangeEvent } from "react";
import { CalendarPageTabOption, CalendarTabsInfo } from "utils/search/objectToSearch";
import { ScheduleShape } from "utils/shape/models/schedule";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type ScheduleUpsertPropsPage = CrudPropsPage & {
    canChangeTab?: never;
    canSetScheduleFor?: never;
    defaultTab?: never;
    isMutate?: never;
}
type ScheduleUpsertPropsDialog = CrudPropsDialog<Schedule> & {
    canChangeTab: boolean;
    canSetScheduleFor: boolean;
    defaultTab: CalendarPageTabOption | `${CalendarPageTabOption}`;
    isMutate: boolean;
};
export type ScheduleUpsertProps = ScheduleUpsertPropsPage | ScheduleUpsertPropsDialog;
export type ScheduleFormProps = FormProps<Schedule, ScheduleShape> & Pick<ScheduleUpsertProps, "canChangeTab" | "canSetScheduleFor" | "isMutate"> & {
    currTab: PageTab<CalendarTabsInfo>;
    handleTabChange: (e: ChangeEvent<unknown> | undefined, tab: PageTab<CalendarTabsInfo>) => unknown;
    tabs: PageTab<CalendarTabsInfo>[];
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
