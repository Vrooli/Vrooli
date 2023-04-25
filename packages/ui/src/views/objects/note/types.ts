import { NoteVersion } from "@local/shared";
import { UpsertProps, ViewProps } from "../types";

export interface NoteUpsertProps extends UpsertProps<NoteVersion> { }
export interface NoteViewProps extends ViewProps<NoteVersion> { }