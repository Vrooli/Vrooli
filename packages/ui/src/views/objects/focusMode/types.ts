import { FocusMode } from "@local/shared";
import { UpsertProps } from "../types";

export type FocusModeUpsertProps = UpsertProps<FocusMode> & {
    partialData?: Partial<FocusMode>;
}
