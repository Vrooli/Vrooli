import { FocusMode, FocusModeShape } from "@local/shared";
import { FormProps } from "forms/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type FocusModeUpsertPropsPage = CrudPropsPage;
type FocusModeUpsertPropsDialog = CrudPropsDialog<FocusMode>;
export type FocusModeUpsertProps = FocusModeUpsertPropsPage | FocusModeUpsertPropsDialog;
export type FocusModeFormProps = FormProps<FocusMode, FocusModeShape>
