import { Comment, CommentThread } from "types";
import { ObjectType } from "utils";

export interface CommentConnectorProps {
    isOpen: boolean;
    objectType: ObjectType['Project'] | ObjectType['Routine'] | ObjectType['Standard'];
    onToggle: () => void;
}

export interface CommentListProps {
    data: CommentThread | null;
    session: Session;
}

export interface CommentListItemProps {
    data: Comment | null;
    loading: boolean;
    session: Session;
}