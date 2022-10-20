import { Comment, CommentThread, NavigableObject } from "types";
import { ObjectType } from "utils";

export interface CommentConnectorProps {
    isOpen: boolean;
    objectType: ObjectType['Project'] | ObjectType['Routine'] | ObjectType['Standard'];
    onToggle: () => void;
}

export interface CommentThreadProps {
    canOpen: boolean;
    data: CommentThread | null;
    language: string;
    session: Session;
    zIndex: number;
}

export interface CommentThreadItemProps {
    data: Comment | null;
    handleCommentAdd: (comment: Comment) => void;
    handleCommentRemove: (comment: Comment) => void;
    isOpen: boolean;
    language: string;
    loading: boolean;
    /**
     * Object which has a comment, not the comment itself or the comment thread
     */
    object: NavigableObject | null | undefined;
    session: Session;
    zIndex;
}