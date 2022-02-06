export interface TitleContainerProps {
    title?: string;
    onClick: () => void;
    loading?: boolean;
    tooltip?: string;
    helpText?: string;
    options?: [string, () => void][];
    sx?: object;
    children: JSX.Element[];
}