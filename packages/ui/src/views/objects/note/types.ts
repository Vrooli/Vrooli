import { NoteVersion, NoteVersionShape } from "@local/shared";
import { FormProps } from "forms/types";
import { CrudProps } from "../types";

export type NoteCrudProps = CrudProps<NoteVersion>;
export type NoteFormProps = FormProps<NoteVersion, NoteVersionShape>
