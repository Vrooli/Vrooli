import { NoteVersion } from "@local/consts";
import { UpsertProps, ViewProps } from "../types";

export interface NoteUpsertProps extends UpsertProps<NoteVersion> { }
export interface NoteViewProps extends ViewProps<NoteVersion> { }
