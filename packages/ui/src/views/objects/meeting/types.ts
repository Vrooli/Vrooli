import { Meeting, MeetingShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type MeetingUpsertPropsPage = CrudPropsPage;
type MeetingUpsertPropsDialog = CrudPropsDialog<Meeting>;
export type MeetingUpsertProps = MeetingUpsertPropsPage | MeetingUpsertPropsDialog;
export type MeetingFormProps = FormProps<Meeting, MeetingShape>
export type MeetingViewProps = ObjectViewProps<Meeting>
