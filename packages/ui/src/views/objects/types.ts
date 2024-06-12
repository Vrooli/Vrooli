import { ListObject, OrArray } from "@local/shared";
import { ObjectViewPropsDialog, ObjectViewPropsPage, ObjectViewPropsPartial } from "views/types";

export type CrudPropsBase = {
    display: "dialog" | "page" | "partial";
    isCreate: boolean;
}
export type CrudPropsPage = CrudPropsBase & ObjectViewPropsPage & {
    display: "page";
    onCancel?: never;
    onClose?: never;
    onCompleted?: never;
    onDeleted?: never;
}
export type CrudPropsDialog<T extends OrArray<ListObject>> = CrudPropsBase & ObjectViewPropsDialog<T> & {
    display: "dialog";
    /** Closes the view and clears cached data */
    onCancel: () => unknown;
    /** Closes the view without clearing cached data */
    onClose: () => unknown;
    /** Closes the view, clears cached data, and returns created/updated data */
    onCompleted: (data: T) => unknown;
    /** Closes the view, clears cached data, and returns deleted object (or objects if arrray) */
    onDeleted: (data: T) => unknown;
}
export type CrudPropsPartial<T extends OrArray<ListObject>> = CrudPropsBase & ObjectViewPropsPartial<T> & {
    display: "partial";
    onCancel?: never;
    onClose?: never;
    onCompleted?: never;
    onDeleted?: never;
}
export type CrudProps<T extends OrArray<ListObject>> = CrudPropsPage | CrudPropsDialog<T> | CrudPropsPartial<T>;
