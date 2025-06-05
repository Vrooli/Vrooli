import { type ApiVersion, type ApiVersionShape } from "@vrooli/shared";
import { type CrudPropsDialog, type CrudPropsPage, type FormProps, type ObjectViewProps } from "../../../types.js";

type ApiUpsertPropsPage = CrudPropsPage;
type ApiUpsertPropsDialog = CrudPropsDialog<ApiVersion>;
type ApiUpsertPropsPartial = CrudPropsPartial<ApiVersion>;
export type ApiUpsertProps = ApiUpsertPropsPage | ApiUpsertPropsDialog | ApiUpsertPropsPartial;
export type ApiFormProps = FormProps<ApiVersion, ApiVersionShape> & {
    versions: string[];
}
export type ApiViewProps = ObjectViewProps<ApiVersion>
