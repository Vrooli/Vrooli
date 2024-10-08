import { ApiVersion, ApiVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type ApiUpsertPropsPage = CrudPropsPage;
type ApiUpsertPropsDialog = CrudPropsDialog<ApiVersion>;
export type ApiUpsertProps = ApiUpsertPropsPage | ApiUpsertPropsDialog;
export type ApiFormProps = FormProps<ApiVersion, ApiVersionShape> & {
    versions: string[];
}
export type ApiViewProps = ObjectViewProps<ApiVersion>
