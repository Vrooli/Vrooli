import { CodeVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { CodeVersionShape } from "utils/shape/models/codeVersion";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type CodeUpsertPropsPage = CrudPropsPage;
type CodeUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
export type CodeUpsertProps = CodeUpsertPropsPage | CodeUpsertPropsDialog;
export type CodeFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type CodeViewProps = ObjectViewProps<CodeVersion>
