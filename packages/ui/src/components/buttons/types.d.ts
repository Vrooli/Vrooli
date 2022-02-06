import { ButtonProps } from '@mui/material';

export interface HelpButtonProps extends ButtonProps {
    id?: string;
    markdown: string;
    sx?: object;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: any
};