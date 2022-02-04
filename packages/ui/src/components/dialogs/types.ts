import { DialogProps, MenuProps } from '@mui/material';
import { HelpButtonProps } from "components/buttons/types";
import { SvgIconComponent } from '@mui/icons-material';
import { ReportFor } from '@local/shared';
import { Organization, Session } from 'types';

export interface AlertDialogProps extends DialogProps {};

export interface ComponentWrapperDialogProps extends DialogProps {};

export interface FormDialogProps {
    title: string;
    children: JSX.Element;
    maxWidth?: string | number;
    onClose: () => void;
}

export interface ListMenuItemData {
    label: string; // Text to display
    value: string; // Value to pass back
    Icon?: SvgIconComponent; // Icon to display
    iconColor?: string; // Color of icon, if different than text
    helpData?: HelpButtonProps; // If set, displays help button with data
}
export interface ListMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    onSelect: (value: any) => void;
    onClose: () => void;
    title?: string;
    data?: ListMenuItemData[];
}

export enum ObjectDialogState {
    Add,
    Edit,
    View,
}
export enum ObjectDialogAction {
    Add,
    Cancel,
    Close,
    Edit,
    Next,
    Previous,
    Save,
}

export interface BaseObjectDialogProps extends DialogProps {
    title: string;
    open: boolean;
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    state: ObjectDialogState; // Determines what options to show
    onAction: (state: ObjectDialogAction) => any; // Callback when option button or close button is pressed
    children: JSX.Element | JSX.Element[];
};

export interface SelectInterestsDialogProps extends DialogProps {
    session: Session;
    open: boolean;
    onClose: () => any;
    showHidden?: boolean; // Show section for tags to hide
}

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

export interface OrganizationDialogProps {
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Organization>;
    session: Session;
};