import { ListObject } from "utils/display/listTools";
import { ObjectViewProps } from "views/types";

export interface UpsertProps<T extends ListObject> extends Omit<ObjectViewProps<T>, "display" | "onClose"> {
    isCreate: boolean;
    onCancel?: () => unknown;
    onCompleted?: (data: T) => unknown;
}

export interface CrudProps<T extends ListObject> extends UpsertProps<T> {
    onDeleted?: (id: string) => unknown;
}
