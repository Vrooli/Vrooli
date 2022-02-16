import { ButtonProps } from '@mui/material';

export interface HelpButtonProps extends ButtonProps {
    id?: string;
    /**
     * Markdown displayed in the popup menu
     */
    markdown: string;
    /**
     * Style applied to the root element
     */
    sxRoot?: object;
    /**
     * Style applied to the question mark icon
     */
    sx?: object;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: any
};