import { Meeting } from "@local/shared";
import { FormProps } from "forms/types";
import { MeetingShape } from "utils/shape/models/meeting";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type MeetingUpsertPropsPage = CrudPropsPage;
type MeetingUpsertPropsDialog = CrudPropsDialog<Meeting>;
export type MeetingUpsertProps = MeetingUpsertPropsPage | MeetingUpsertPropsDialog;
export type MeetingFormProps = FormProps<Meeting, MeetingShape>
export type MeetingViewProps = ObjectViewProps<Meeting>
