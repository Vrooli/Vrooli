import { Comment } from "types";

export interface CommentListItemProps {
    data: Comment | null;
    loading: boolean;
    session: Session;
    onVote: (isUpvote: boolean | null) => void;
}