import { User } from "@local/shared";
import { LinearProgressProps } from "@mui/material";
import { PageTab } from "hooks/useTabs";
import { SxType } from "types";
import { ChatMessageShape } from "utils/shape/models/chatMessage";

export interface ChatBubbleProps {
    activeIndex: number;
    chatWidth: number;
    message: ChatMessageShape;
    messagesCount: number;
    isOwn: boolean;
    onActiveIndexChange: (index: number) => unknown;
    onDeleted: (message: ChatMessageShape) => unknown;
    onReply: (message: ChatMessageShape) => unknown;
    onRetry: (message: ChatMessageShape) => unknown;
    onUpdated: (message: ChatMessageShape) => unknown;
}

export interface ChatBubbleTreeProps {
    chatWidth: number;
    message: ChatMessageShape;
    index: number;
    isOwn: boolean;
    onDeleted: (message: ChatMessageShape) => unknown;
    onUpdated: (message: ChatMessageShape) => unknown;
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
