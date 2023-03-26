import { BaseViewProps } from "views/types";

export interface CreateProps<T> extends BaseViewProps {
    onCancel: () => any;
    onCreated: (data: T) => any;
    zIndex?: number;
}
export interface UpdateProps<T> extends BaseViewProps {
    onCancel: () => any;
    onUpdated: (data: T) => any;
    zIndex?: number;
}
export interface ViewProps<T> extends BaseViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
    zIndex?: number;
}