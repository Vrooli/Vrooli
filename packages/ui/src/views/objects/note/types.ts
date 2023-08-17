import { NoteVersion } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type NoteUpsertProps = UpsertProps<NoteVersion>
export type NoteViewProps = ObjectViewProps<NoteVersion>
