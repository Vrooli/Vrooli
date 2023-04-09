import { BaseViewProps } from "views/types";

export interface ViewProps<T> extends BaseViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
    zIndex?: number;
}

export interface UpsertProps<T> extends BaseViewProps {
    isCreate: boolean;
    onCancel?: () => any;
    onCompleted?: (data: T) => any;
    zIndex?: number;
}