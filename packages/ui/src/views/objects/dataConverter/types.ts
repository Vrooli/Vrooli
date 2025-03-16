import { CodeVersion, CodeVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types.js";

type DataConverterUpsertPropsPage = CrudPropsPage;
type DataConverterUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
export type DataConverterUpsertProps = DataConverterUpsertPropsPage | DataConverterUpsertPropsDialog;
export type DataConverterFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type DataConverterViewProps = ObjectViewProps<CodeVersion>
