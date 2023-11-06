import { OrArray } from "@local/shared";
import { ListObject } from "utils/display/listTools";
import { ObjectViewProps } from "views/types";

export interface UpsertProps<T extends OrArray<ListObject>> extends Omit<ObjectViewProps<T>, "display" | "onClose"> {
    isCreate: boolean;
    /** Closes the view and clears cached data */
    onCancel?: () => unknown;
    /** Closes the view without clearing cached data */
    onClose?: () => unknown;
    /** Closes the view, clears cached data, and returns created/updated data */
    onCompleted?: (data: T) => unknown;
}

export interface CrudProps<T extends OrArray<ListObject>> extends UpsertProps<T> {
    onDeleted?: (id: string) => unknown;
}
