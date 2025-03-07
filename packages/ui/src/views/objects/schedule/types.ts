import { Schedule, ScheduleShape, TranslationKeyCommon } from "@local/shared";
import { SvgComponent } from "types";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types.js";

export type ScheduleForType = "FocusMode" | "Meeting" | "RunProject" | "RunRoutine";

export type ScheduleForOption = {
    Icon: SvgComponent
    labelKey: TranslationKeyCommon;
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
