import { ApiVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type ApiUpsertPropsPage = CrudPropsPage;
type ApiUpsertPropsDialog = CrudPropsDialog<ApiVersion>;
export type ApiUpsertProps = ApiUpsertPropsPage | ApiUpsertPropsDialog;
export type ApiFormProps = FormProps<ApiVersion, ApiVersionShape> & {
    versions: string[];
}
export type ApiViewProps = ObjectViewProps<ApiVersion>
