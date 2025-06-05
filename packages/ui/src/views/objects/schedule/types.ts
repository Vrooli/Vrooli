import { type Schedule, type ScheduleShape, type TranslationKeyCommon } from "@vrooli/shared";
import { type IconInfo } from "../../../icons/Icons.js";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

export type ScheduleForType = "Meeting" | "Run";

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
    pathname: string;
    scheduleFor: ScheduleForOption | null;
    handleScheduleForChange: (scheduleFor: ScheduleForOption) => unknown;
}
export type ScheduleViewProps = ObjectViewProps<Schedule>
