import { NoteVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { CrudProps } from "../types";

export type NoteCrudProps = CrudProps<NoteVersion>;
export type NoteFormProps = FormProps<NoteVersion, NoteVersionShape>
