import { DialogProps, MenuProps } from '@mui/material';
import { HelpButtonProps } from "components/buttons/types";
import { SvgIconComponent } from '@mui/icons-material';

export interface AlertDialogProps extends DialogProps {};

export interface ComponentWrapperDialogProps extends DialogProps {};

export interface FormDialogProps {
    title: string;
    children: JSX.Element;
    maxWidth?: string | number;
    onClose: () => void;
}

export interface ListMenuItemData {
    label: string;
    value: string;
    Icon?: SvgIconComponent;
    helpData?: HelpButtonProps;
}
export interface ListMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    onSelect: (value: any) => void;
    onClose: () => void;
    title?: string;
    data?: ListMenuItemData[];
}

export interface NewProjectDialogProps extends DialogProps {
    open?: boolean;
    onClose: () => any;
};