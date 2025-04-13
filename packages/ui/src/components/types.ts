/* c8 ignore start */
import { ChatMessageShape, Comment, CommentThread, NavigableObject } from "@local/shared";
import { LinearProgressProps } from "@mui/material";
import { ReactNode } from "react";
import { PageTab } from "../hooks/useTabs.js";
import { SxType } from "../types.js";
import { TabListType } from "../utils/search/objectToSearch.js";

export interface ChatBubbleProps {
    /** Which sibling (version) is currently being displayed */
    activeIndex: number;
    chatWidth: number;
    message: ChatMessageShape;
    /** The number of siblings (versions) this message has, including itself */
    numSiblings: number;
    /** If the chat is only between you and other bots */
    isBotOnlyChat: boolean;
    /** Is a message you created */
    isOwn: boolean;
    /** 
     * Change the active index (sibling/version) being displayed.
     * This also changes which messages are displayed below, as 
     * versions branch the message tree.
     */
    onActiveIndexChange: (index: number) => unknown;
    onDeleted: (message: ChatMessageShape) => unknown;
    onEdit: (originalMessage: ChatMessageShape) => unknown;
    onReply: (message: ChatMessageShape) => unknown;
    /** Regenerate a bot response */
    onRegenerateResponse: (message: ChatMessageShape) => unknown;
    /** Try to send a failed message */
    onRetry: (message: ChatMessageShape) => unknown;
}

export interface CommentConnectorProps {
    isOpen: boolean;
    parentType: "User" | "Team";
    onToggle: () => unknown;
    owner?: any;
}

export interface CommentThreadProps {
    canOpen: boolean;
    data: CommentThread | null;
    language: string;
}

export interface CommentThreadItemProps {
    data: Comment | null;
    handleCommentRemove: (comment: Comment) => unknown;
    handleCommentUpsert: (comment: Comment) => unknown;
    isOpen: boolean;
    language: string;
    loading: boolean;
    /** Object which has a comment, not the comment itself or the comment thread */
    object: NavigableObject | null | undefined;
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

export interface PageTabsProps<TabList extends TabListType = TabListType> {
    ariaLabel: string,
    currTab: PageTab<TabList[number]>,
    fullWidth?: boolean,
    id?: string,
    /** Ignore Icons in tabs, rendering them using labels instead */
    ignoreIcons?: boolean,
    onChange: (event: React.ChangeEvent<unknown>, value: PageTab<TabList[number]>) => unknown,
    tabs: PageTab<TabList[number]>[],
    sx?: SxType,
}

export interface SlideProps {
    id: string;
    children: ReactNode;
    sx?: SxType;
}
