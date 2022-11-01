import { SnackSeverity } from './BasicSnack/BasicSnack';

export interface BasicSnackProps {
    buttonClicked?: (event?: any) => any;
    buttonText?: string;
    /**
     * Anything you'd like to log in development mode
     */
    data?: any;
    duration?: number | 'persist';
    handleClose: () => any;
    id: string;
    message?: string;
    severity?: SnackSeverity;
}

export interface CookiesSnackProps {
    handleClose: () => any;
}