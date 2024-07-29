import { NoteVersion, NoteVersionShape } from "@local/shared";
import { CrudProps, FormProps } from "../../../types";

export type NoteCrudProps = CrudProps<NoteVersion>;
export type NoteFormProps = FormProps<NoteVersion, NoteVersionShape>
