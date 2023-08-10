import { Meeting } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type MeetingUpsertProps = UpsertProps<Meeting>
export type MeetingViewProps = ObjectViewProps<Meeting>
