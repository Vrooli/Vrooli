import { LlmTaskInfo, User } from "@local/shared";
import { LinearProgressProps } from "@mui/material";
import { PageTab } from "hooks/useTabs";
import { SxType } from "types";
import { TabsInfo } from "utils/search/objectToSearch";
import { ChatMessageShape } from "utils/shape/models/chatMessage";

export interface ChatBubbleProps {
    /** Which sibling (version) is currently being displayed */
    activeIndex: number;
    chatWidth: number;
    message: ChatMessageShape;
    /** The number of siblings (versions) this message has, including itself */
    numSiblings: number;
    /** If the chat is only between you and other bots */
    isBotOnlyChat: boolean;
    /** Is the most recent message (or one of the versions of the most recent message) */
    isLastMessage: boolean;
    /** Is a message you created */
    isOwn: boolean;
    /** 
     * Change the active index (sibling/version) being displayed.
     * This also changes which messages are displayed below, as 
     * versions branch the message tree.
     */
    onActiveIndexChange: (index: number) => unknown;
    onDeleted: (message: ChatMessageShape) => unknown;
    onReply: (message: ChatMessageShape) => unknown;
    onRetry: (message: ChatMessageShape) => unknown;
    onTaskClick: (task: LlmTaskInfo) => unknown;
    onUpdated: (message: ChatMessageShape) => unknown;
    tasks: LlmTaskInfo[];
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

export interface PageTabsProps<TabList extends TabsInfo> {
    ariaLabel: string,
    currTab: PageTab<TabList>,
    fullWidth?: boolean,
    id?: string,
    /** Ignore Icons in tabs, rendering them using labels instead */
    ignoreIcons?: boolean,
    onChange: (event: React.ChangeEvent<unknown>, value: PageTab<TabList>) => unknown,
    tabs: PageTab<TabList>[],
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
