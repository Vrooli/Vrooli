import { Comment } from "@local/shared";
import { Stack } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { updateArray } from "utils/shape/general";
import { CommentConnector } from "../CommentConnector/CommentConnector";
import { CommentThreadItem } from "../CommentThreadItem/CommentThreadItem";
import { CommentThreadProps } from "../types";

/**
 * Comment and its list of child comments (which can have their own children and so on). 
 * Each level  contains a list of comment items, then a "Show more" text button.
 * To the left of this is a CommentConnector item, which is a collapsible line.
 */
export const CommentThread = ({
    canOpen,
    data,
    language,
    zIndex,
}: CommentThreadProps) => {
    // open state
    const [isOpen, setIsOpen] = useState(true);

    const [childData, setChildData] = useState(data?.childThreads ?? []);
    useMemo(() => {
        setChildData(data?.childThreads ?? []);
    }, [data]);
    const removeComment = useCallback((comment: Comment) => {
        setChildData(curr => [...curr.filter(c => c.comment.id !== comment.id)]);
    }, []);
    const upsertComment = useCallback((comment: Comment) => {
        // Find index of comment
        const index = childData.findIndex(c => c.comment.id === comment.id);
        // If not found, must be a new comment
        if (index !== -1) {
            // Make comment first, so you can see it without having to scroll to the bottom
            setChildData(curr => [{
                __typename: "CommentThread",
                comment: comment as any,
                childThreads: [],
                endCursor: null,
                totalInThread: 0,
            }, ...curr]);
        }
        // If found, update comment
        else {
            setChildData(curr => updateArray(curr, index, {
                ...curr[index],
                comment,
            }));
        }
    }, [childData]);

    // list of comment items
    const children = useMemo(() => {
        if (!data) return [];
        return childData.map((child, index) => {
            return <CommentThread
                key={`thread-${data.comment.id}-${index}`}
                canOpen={canOpen && isOpen}
                data={{
                    ...child,
                    childThreads: [],
                }}
                language={language}
                zIndex={zIndex}
            />;
        });
    }, [canOpen, childData, data, isOpen, language, zIndex]);

    return data && canOpen ? (
        <Stack direction="row" spacing={1} pl={2} pr={2}>
            {/* Comment connector */}
            <CommentConnector
                isOpen={isOpen}
                parentType={data.comment.owner?.__typename ?? "User"}
                onToggle={() => setIsOpen(!isOpen)}
            />
            {/* Comment and child comments */}
            <Stack direction="column" spacing={1} sx={{ width: "100%" }}>
                {/* Comment */}
                <CommentThreadItem
                    data={data.comment}
                    handleCommentRemove={removeComment}
                    handleCommentUpsert={upsertComment}
                    isOpen={isOpen}
                    language={language}
                    loading={false}
                    object={data.comment.commentedOn}
                    zIndex={zIndex}
                />
                {/* Child comments */}
                {children}
                {/* "Show More" button */}
                {/* TODO */}
            </Stack>
        </Stack>
    ) : null;
};
