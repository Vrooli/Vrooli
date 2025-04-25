import { Schedule, ScheduleShape, TranslationKeyCommon } from "@local/shared";
import { IconInfo } from "../../../icons/Icons.js";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

export type ScheduleForType = "Meeting" | "RunProject" | "RunRoutine";

export type ScheduleForOption = {
    iconInfo: IconInfo;
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
type ScheduleUpsertPropsPartial = CrudPropsPartial<Schedule> & {
    canSetScheduleFor?: never;
    defaultScheduleFor?: never;
    isMutate?: never;
}
export type ScheduleUpsertProps = ScheduleUpsertPropsPage | ScheduleUpsertPropsDialog | ScheduleUpsertPropsPartial;
export type ScheduleFormProps = FormProps<Schedule, ScheduleShape> & Pick<ScheduleUpsertProps, | "canSetScheduleFor" | "isMutate"> & {
    scheduleFor: ScheduleForOption | null;
    handleScheduleForChange: (scheduleFor: ScheduleForOption) => unknown;
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
