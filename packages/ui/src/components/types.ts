import { ChatMessage, SvgComponent, User } from "@local/shared";
import { LinearProgressProps } from "@mui/material";

export interface ChatBubbleProps {
    message: ChatMessage & { isUnsent?: boolean }
    index: number;
    isOwn: boolean;
    onUpdated: (message: ChatMessage & { isUnsent: boolean }) => void;
    zIndex: number;
}

export interface ChatBubbleStatusProps {
    isEditing: boolean;
    /** Indicates if the message is still sending */
    isSending: boolean;
    /** Indicates if there has been an error in sending the message */
    hasError: boolean;
    onEdit: () => void;
    onRetry: () => void;
}


export interface CompletionBarProps extends Omit<LinearProgressProps, "value"> {
    isLoading?: boolean;
    showLabel?: boolean;
    value: number;
}

export interface DiagonalWaveLoaderProps {
    size?: number;
    color?: string;
    sx?: { [key: string]: any };
}

export type PageTab<T> = {
    color?: string,
    href?: string,
    /**
     * If set, icon is displayed and label becomes a toolip
     */
    Icon?: SvgComponent;
    index: number,
    label: string,
    value: T
};

export interface PageTabsProps<T> {
    ariaLabel: string,
    currTab: PageTab<T>,
    fullWidth?: boolean,
    id?: string,
    onChange: (event: React.ChangeEvent<unknown>, value: any) => void,
    tabs: PageTab<T>[],
}

export interface ProfileGroupProps {
    sx?: { [key: string]: any };
    users: User[];
}

export interface TwinklingStarsProps {
    amount?: number;
    size?: number;
    color?: string;
    speed?: number;
    sx?: { [key: string]: any };
}
