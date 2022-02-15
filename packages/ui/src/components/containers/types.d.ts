export interface TitleContainerProps {
    id?: string;
    title?: string;
    onClick?: () => void;
    loading?: boolean;
    tooltip?: string;
    helpText?: string;
    options?: [string, () => void][];
    sx?: object;
    children: JSX.Element | JSX.Element[];
}

// label, Icon, disabled, isSubmit, onClick
export type DialogActionItem = [string, any, boolean, boolean, () => void,]

export interface DialogActionsContainerProps {
    fixed?: boolean; // if true, the actions will be fixed to the bottom of the window
    actions: DialogActionItem[];
    onResize?: ({ height: number, width: number }) => any;
}