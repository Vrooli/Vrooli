import { CodeVersion, CodeVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type DataConverterUpsertPropsPage = CrudPropsPage;
type DataConverterUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
type DataConverterUpsertPropsPartial = CrudPropsPartial<CodeVersion>;
export type DataConverterUpsertProps = DataConverterUpsertPropsPage | DataConverterUpsertPropsDialog | DataConverterUpsertPropsPartial;
export type DataConverterFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type DataConverterViewProps = ObjectViewProps<CodeVersion>
