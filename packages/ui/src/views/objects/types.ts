import { OrArray } from "@local/shared";
import { ListObject } from "utils/display/listTools";
import { ObjectViewProps } from "views/types";

type UpsertPropsBase<T extends OrArray<ListObject>> = Omit<ObjectViewProps<T>, "display" | "onClose"> & {
    isCreate: boolean;
}
type UpsertPropsPage<T extends OrArray<ListObject>> = UpsertPropsBase<T> & {
    onCancel: undefined;
    onClose: undefined;
    onCompleted: undefined;
}
type UpsertPropsDialog<T extends OrArray<ListObject>> = UpsertPropsBase<T> & {
    /** Closes the view and clears cached data */
    onCancel: () => unknown;
    /** Closes the view without clearing cached data */
    onClose: () => unknown;
    /** Closes the view, clears cached data, and returns created/updated data */
    onCompleted: (data: T) => unknown;
}
export type UpsertProps<T extends OrArray<ListObject>> = UpsertPropsPage<T> | UpsertPropsDialog<T>;

type CrudPropsBase<T extends OrArray<ListObject>> = UpsertProps<T>;
type CrudPropsPage<T extends OrArray<ListObject>> = CrudPropsBase<T> & {
    onDeleted: undefined;
}
type CrudPropsDialog<T extends OrArray<ListObject>> = CrudPropsBase<T> & {
    onDeleted: (id: string) => unknown;
}
export type CrudProps<T extends OrArray<ListObject>> = CrudPropsPage<T> | CrudPropsDialog<T>;
