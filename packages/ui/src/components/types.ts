import { ChatMessage, User } from "@local/shared";
import { LinearProgressProps } from "@mui/material";
import { SvgComponent, SxType } from "types";

export interface ChatBubbleProps {
    message: ChatMessage & { isUnsent?: boolean }
    index: number;
    isOwn: boolean;
    onUpdated: (message: ChatMessage & { isUnsent: boolean }) => unknown;
    zIndex: number;
}

export interface ChatBubbleStatusProps {
    isEditing: boolean;
    /** Indicates if the message is still sending */
    isSending: boolean;
    /** Indicates if there has been an error in sending the message */
    hasError: boolean;
    onEdit: () => unknown;
    onRetry: () => unknown;
}


export interface CompletionBarProps extends Omit<LinearProgressProps, "value"> {
    isLoading?: boolean;
    showLabel?: boolean;
    value: number;
}

export interface DiagonalWaveLoaderProps {
    size?: number;
    color?: string;
    sx?: SxType;
}

export type PageTab<T> = {
    color?: string,
    href?: string,
    /** If set, icon is displayed and label becomes a toolip */
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
    /** Ignore Icons in tabs, rendering them using labels instead */
    ignoreIcons?: boolean,
    onChange: (event: React.ChangeEvent<unknown>, value: any) => unknown,
    tabs: PageTab<T>[],
    sx?: SxType,
}

export interface ProfileGroupProps {
    sx?: SxType;
    users: User[];
}

export interface TwinklingStarsProps {
    amount?: number;
    size?: number;
    color?: string;
    speed?: number;
    sx?: SxType;
}
