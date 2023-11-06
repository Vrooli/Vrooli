import { StandardVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type StandardUpsertProps = UpsertProps<StandardVersion>
export type StandardFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type StandardViewProps = ObjectViewProps<StandardVersion>
