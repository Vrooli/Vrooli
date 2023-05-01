import { Comment, CommentThread } from "@local/shared";
import { NavigableObject } from "types";

export interface CommentConnectorProps {
    isOpen: boolean;
    parentType: 'User' | 'Organization';
    onToggle: () => void;
}

export interface CommentThreadProps {
    canOpen: boolean;
    data: CommentThread | null;
    language: string;
    zIndex: number;
}

export interface CommentThreadItemProps {
    data: Comment | null;
    handleCommentRemove: (comment: Comment) => void;
    handleCommentUpsert: (comment: Comment) => void;
    isOpen: boolean;
    language: string;
    loading: boolean;
    /**
     * Object which has a comment, not the comment itself or the comment thread
     */
    object: NavigableObject | null | undefined;
    zIndex;
}