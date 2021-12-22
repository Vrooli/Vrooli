import { DialogProps } from "@material-ui/core";

export interface AlertDialogProps extends DialogProps {};

export interface ComponentWrapperDialogProps extends DialogProps {};

export interface ListDialogItemData {
    label?: string;
    value?: string;
}
export interface ListDialogProps extends DialogProps {
    open?: boolean;
    onClose: (value?: string) => void;
    title?: string;
    data?: ListDialogItemData[];
}

export interface NewProjectDialogProps extends DialogProps {
    open?: boolean;
    onClose: () => any;
};