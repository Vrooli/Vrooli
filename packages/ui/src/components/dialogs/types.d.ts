import { DialogProps, MenuProps } from '@mui/material';
import { HelpButtonProps } from "components/buttons/types";
import { SvgIconComponent } from '@mui/icons-material';
import { ReportFor } from '@local/shared';

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
export interface AddDialogBaseProps extends DialogProps {
    title: string;
    open: boolean;
    onSubmit: (value: any) => any;
    onClose: () => any;
    children: JSX.Element | JSX.Element[];
};

export interface ViewDialogBaseProps extends DialogProps {
    title: string;
    open: boolean;
    canEdit?: boolean; // Can only edit if you own the object
    isEditing?: boolean; // Is currently editing
    onEdit?: () => any; // Callback when starting to edit object
    onSave?: () => any; // Callback when saving changes
    onRevert?: () => any; // Callback when reverting changes
    onClose?: () => any; // Callback when closing dialog
    children: JSX.Element | JSX.Element[];
};

export interface ShareDialogProps extends DialogProps {
    open: boolean;
    onClose: () => any;
}

export interface ReportDialogProps extends DialogProps {
    open: boolean;
    onClose: () => any;
    title?: string;
    reportFor: ReportFor;
    forId: string;
}