import { PartialWithType } from "types";
import { ListObject } from "utils/display/listTools";
import { BaseViewProps } from "views/types";

export interface ViewProps<T extends ListObject> extends BaseViewProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: PartialWithType<T>;
}

export interface UpsertProps<T> extends Omit<BaseViewProps, "onClose"> {
    isCreate: boolean;
    onCancel?: () => unknown;
    onCompleted?: (data: T) => unknown;
}
