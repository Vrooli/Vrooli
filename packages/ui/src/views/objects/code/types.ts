import { CodeVersion, CodeVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type CodeUpsertPropsPage = CrudPropsPage;
type CodeUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
export type CodeUpsertProps = CodeUpsertPropsPage | CodeUpsertPropsDialog;
export type CodeFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type CodeViewProps = ObjectViewProps<CodeVersion>
