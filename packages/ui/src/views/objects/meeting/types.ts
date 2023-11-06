import { Meeting } from "@local/shared";
import { FormProps } from "forms/types";
import { MeetingShape } from "utils/shape/models/meeting";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type MeetingUpsertProps = UpsertProps<Meeting>
export type MeetingFormProps = FormProps<Meeting, MeetingShape>
export type MeetingViewProps = ObjectViewProps<Meeting>
