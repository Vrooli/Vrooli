import { SnackSeverity } from "./BasicSnack/BasicSnack";

export interface BasicSnackProps {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: any) => any;
    buttonText?: string;
    /**
     * Anything you'd like to log in development mode
     */
    data?: any;
    handleClose: () => any;
    id: string;
    message?: string;
    severity?: SnackSeverity | `${SnackSeverity}`;
}

export interface CookiesSnackProps {
    handleClose: () => any;
}
