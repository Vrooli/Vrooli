import { StandardVersion, StandardVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type PromptUpsertPropsPage = CrudPropsPage;
type PromptUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
export type PromptUpsertProps = PromptUpsertPropsPage | PromptUpsertPropsDialog;
export type PromptFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type PromptViewProps = ObjectViewProps<StandardVersion>
