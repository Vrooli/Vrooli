import { Schedule } from "@local/shared";
import { FormProps } from "forms/types";
import { PageTab } from "hooks/useTabs";
import { ChangeEvent } from "react";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
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
    defaultTab: CalendarPageTabOption;
    isMutate: boolean;
};
export type ScheduleUpsertProps = ScheduleUpsertPropsPage | ScheduleUpsertPropsDialog;
export type ScheduleFormProps = FormProps<Schedule, ScheduleShape> & Pick<ScheduleUpsertProps, "canChangeTab" | "canSetScheduleFor" | "isMutate"> & {
    currTab: PageTab<CalendarPageTabOption, true>;
    handleTabChange: (e: ChangeEvent<unknown> | undefined, tab: PageTab<CalendarPageTabOption, true>) => unknown;
    tabs: PageTab<CalendarPageTabOption, true>[];
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
