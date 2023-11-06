import { ApiVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type ApiUpsertProps = UpsertProps<ApiVersion>
export type ApiFormProps = FormProps<ApiVersion, ApiVersionShape> & {
    versions: string[];
}
export type ApiViewProps = ObjectViewProps<ApiVersion>
