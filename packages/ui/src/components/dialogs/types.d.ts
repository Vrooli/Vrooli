import { DialogProps } from "@material-ui/core";
import { OverridableComponent } from '@material-ui/core/OverridableComponent';
import { HelpButtonProps } from "components/buttons/types";

export interface AlertDialogProps extends DialogProps {};

export interface ComponentWrapperDialogProps extends DialogProps {};

export interface ListDialogItemData {
    label: string;
    value: string;
    Icon?: OverridableComponent<SvgIconTypeMap<{}, "svg">>;
    helpData?: HelpButtonProps;
}
export interface ListDialogProps extends DialogProps {
    open?: boolean;
    onSelect: (value: string) => void;
    onClose: () => void;
    title?: string;
    data?: ListDialogItemData[];
}

export interface NewProjectDialogProps extends DialogProps {
    open?: boolean;
    onClose: () => any;
};