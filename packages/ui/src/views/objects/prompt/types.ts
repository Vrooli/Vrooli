import { type StandardVersion, type StandardVersionShape } from "@vrooli/shared";
import { type CrudPropsDialog, type CrudPropsPage, type FormProps, type ObjectViewProps } from "../../../types.js";

type PromptUpsertPropsPage = CrudPropsPage;
type PromptUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
export type PromptUpsertProps = PromptUpsertPropsPage | PromptUpsertPropsDialog;
export type PromptFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type PromptViewProps = ObjectViewProps<StandardVersion>
