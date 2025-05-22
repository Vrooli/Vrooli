import { type CodeVersion, type CodeVersionShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

type DataConverterUpsertPropsPage = CrudPropsPage;
type DataConverterUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
type DataConverterUpsertPropsPartial = CrudPropsPartial<CodeVersion>;
export type DataConverterUpsertProps = DataConverterUpsertPropsPage | DataConverterUpsertPropsDialog | DataConverterUpsertPropsPartial;
export type DataConverterFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type DataConverterViewProps = ObjectViewProps<CodeVersion>
