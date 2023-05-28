import { BaseViewProps } from "views/types";

export interface ViewProps<T> extends BaseViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<T>;
}

export interface UpsertProps<T> extends Omit<BaseViewProps, "onClose"> {
    isCreate: boolean;
    onCancel?: () => any;
    onCompleted?: (data: T) => any;
}
