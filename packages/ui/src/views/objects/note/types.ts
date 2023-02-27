import { NoteVersion } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface NoteCreateProps extends CreateProps<NoteVersion> {}
export interface NoteUpdateProps extends UpdateProps<NoteVersion> {}
export interface NoteViewProps extends ViewProps<NoteVersion> {}