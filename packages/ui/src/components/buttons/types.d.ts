import { ButtonProps } from '@mui/material';
import { SvgProps } from 'assets/img/types';
import { Status } from 'utils';

export interface GridSubmitButtonsProps {
    disabledCancel?: boolean;
    disabledSubmit?: boolean;
    errors?: { [key: string]: string };
    isCreate: boolean;
    onCancel: () => void;
    onSetSubmitting?: (isSubmitting: boolean) => void;
    onSubmit?: () => void;
}

export interface HelpButtonProps extends ButtonProps {
    id?: string;
    /**
     * Markdown displayed in the popup menu
     */
    markdown: string;
    /**
     * On click event. Not needed to open the menu
     */
    onClick?: (event: React.MouseEvent) => void;
    /**
     * Style applied to the root element
     */
    sxRoot?: object;
    /**
     * Style applied to the question mark icon
     */
    sx?: SvgProps;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: any
};


export interface StatusMessageArray {
    status: Status;
    messages: string[];
}

export interface StatusButtonProps extends ButtonProps {
    status: Status;
    messages: string[];
}