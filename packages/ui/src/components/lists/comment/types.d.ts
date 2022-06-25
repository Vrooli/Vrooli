import { Comment, CommentThread } from "types";
import { ObjectType } from "utils";

export interface CommentConnectorProps {
    isOpen: boolean;
    objectType: ObjectType['Project'] | ObjectType['Routine'] | ObjectType['Standard'];
    onToggle: () => void;
}

export interface CommentThreadProps {
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
     * ID of the object which has a comment, not
     * the comment itself or its parent
     */
    objectId: string;
    /**
     * Type of the object which has a comment, not
     * the comment itself or its parent
     */
    objectType: ObjectType['Project'] | ObjectType['Routine'] | ObjectType['Standard'];
    session: Session;
    zIndex;
}