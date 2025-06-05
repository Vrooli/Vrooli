import { type Meeting, type MeetingShape } from "@vrooli/shared";
import { type CrudPropsDialog, type CrudPropsPage, type FormProps, type ObjectViewProps } from "../../../types.js";

type MeetingUpsertPropsPage = CrudPropsPage;
type MeetingUpsertPropsDialog = CrudPropsDialog<Meeting>;
export type MeetingUpsertProps = MeetingUpsertPropsPage | MeetingUpsertPropsDialog;
export type MeetingFormProps = FormProps<Meeting, MeetingShape>
export type MeetingViewProps = ObjectViewProps<Meeting>
