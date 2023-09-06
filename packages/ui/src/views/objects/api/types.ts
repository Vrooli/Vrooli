import { ApiVersion } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type ApiUpsertProps = UpsertProps<ApiVersion>
export type ApiViewProps = ObjectViewProps<ApiVersion>
