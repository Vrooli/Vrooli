import { ChatMessage, User } from "@local/shared";
import { LinearProgressProps } from "@mui/material";
import { PageTab } from "hooks/useTabs";
import { SxType } from "types";

export interface ChatBubbleProps {
    message: ChatMessage & { isUnsent?: boolean }
    index: number;
    isOwn: boolean;
    onUpdated: (message: ChatMessage & { isUnsent: boolean }) => unknown;
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
    sxs?: {
        root?: SxType;
        bar?: SxType;
        barBox?: SxType;
        label?: SxType;
    }
    value: number;
}

export interface DiagonalWaveLoaderProps {
    size?: number;
    color?: string;
    sx?: SxType;
}

export interface PageTabsProps<T, S extends boolean = true> {
    ariaLabel: string,
    currTab: PageTab<T, S>,
    fullWidth?: boolean,
    id?: string,
    /** Ignore Icons in tabs, rendering them using labels instead */
    ignoreIcons?: boolean,
    onChange: (event: React.ChangeEvent<unknown>, value: PageTab<T, S>) => unknown,
    tabs: PageTab<T, S>[],
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
