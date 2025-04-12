import { FocusMode, FocusModeShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps } from "../../../types.js";

type FocusModeUpsertPropsPage = CrudPropsPage;
type FocusModeUpsertPropsDialog = CrudPropsDialog<FocusMode>;
export type FocusModeUpsertProps = FocusModeUpsertPropsPage | FocusModeUpsertPropsDialog;
export type FocusModeFormProps = FormProps<FocusMode, FocusModeShape>
