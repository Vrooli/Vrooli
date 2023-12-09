import { FocusMode } from "@local/shared";
import { FormProps } from "forms/types";
import { FocusModeShape } from "utils/shape/models/focusMode";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type FocusModeUpsertPropsPage = CrudPropsPage;
type FocusModeUpsertPropsDialog = CrudPropsDialog<FocusMode>;
export type FocusModeUpsertProps = FocusModeUpsertPropsPage | FocusModeUpsertPropsDialog;
export type FocusModeFormProps = FormProps<FocusMode, FocusModeShape>
