import { Comment, CommentThread, NavigableObject } from "@local/shared";

export interface CommentConnectorProps {
    isOpen: boolean;
    parentType: "User" | "Organization";
    onToggle: () => unknown;
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
