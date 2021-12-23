import { ButtonProps } from '@mui/material';

export interface HelpButtonProps extends ButtonProps {
    title?: string;
    description?: string;
    id?: string;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: any
};