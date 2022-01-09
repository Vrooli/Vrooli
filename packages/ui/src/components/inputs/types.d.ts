export interface DropzoneProps {
    acceptedFileTypes?: string[];
    dropzoneText?: string;
    onUpload: (files: any[]) => any;
    showThumbs?: boolean;
    maxFiles?: number;
    uploadText?: string;
    cancelText?: string;
    disabled?: boolean;
}

export interface SearchBarProps {
    label?: string;
    value: string;
    onChange: (updatedText: string) => any;
    debounce?: number;
    fullWidth?: boolean;
    className?: string;
}

export interface SelectorProps {
    options: any[];
    selected: any;
    handleChange: (change: any) => any;
    fullWidth?: boolean;
    multiple?: boolean;
    inputAriaLabel?: string;
    noneOption?: boolean;
    label?: string;
    required?: boolean;
    disabled?: boolean;
    color?: string;
    className?: string;
    style?: any;
}