import { CommonKey, Schedule, ScheduleShape } from "@local/shared";
import { FormProps } from "forms/types";
import { SvgComponent } from "types";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

export type ScheduleForType = "FocusMode" | "Meeting" | "RunProject" | "RunRoutine";

export type ScheduleForOption = {
    Icon: SvgComponent
    labelKey: CommonKey;
    objectType: ScheduleForType;
}

type ScheduleUpsertPropsPage = CrudPropsPage & {
    canSetScheduleFor?: never;
    defaultScheduleFor?: never;
    isMutate?: never;
}
type ScheduleUpsertPropsDialog = CrudPropsDialog<Schedule> & {
    canSetScheduleFor: boolean;
    defaultScheduleFor: ScheduleForType;
    isMutate: boolean;
};
export type ScheduleUpsertProps = ScheduleUpsertPropsPage | ScheduleUpsertPropsDialog;
export type ScheduleFormProps = FormProps<Schedule, ScheduleShape> & Pick<ScheduleUpsertProps, | "canSetScheduleFor" | "isMutate"> & {
    scheduleFor: ScheduleForOption | null;
    handleScheduleForChange: (scheduleFor: ScheduleForOption) => unknown;
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
