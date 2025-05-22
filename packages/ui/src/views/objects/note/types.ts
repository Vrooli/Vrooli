import { type NoteVersion, type NoteVersionShape } from "@local/shared";
import { type CrudProps, type FormProps } from "../../../types.js";

export type NoteCrudProps = CrudProps<NoteVersion>;
export type NoteFormProps = FormProps<NoteVersion, NoteVersionShape>
