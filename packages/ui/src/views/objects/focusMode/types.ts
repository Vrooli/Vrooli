import { FocusMode } from "@local/shared";
import { FormProps } from "forms/types";
import { FocusModeShape } from "utils/shape/models/focusMode";
import { UpsertProps } from "../types";

export type FocusModeUpsertProps = UpsertProps<FocusMode>;
export type FocusModeFormProps = FormProps<FocusMode, FocusModeShape>
