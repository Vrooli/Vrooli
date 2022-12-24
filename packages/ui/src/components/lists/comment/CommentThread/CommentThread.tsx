import { Stack } from "@mui/material";
import { Comment } from "@shared/consts";
import { useCallback, useMemo, useState } from "react";
import { updateArray } from "utils";
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
    session,
    zIndex,
}: CommentThreadProps) => {
    // open state
    const [isOpen, setIsOpen] = useState(true);

    const [childData, setChildData] = useState(data?.childThreads ?? []);
    useMemo(() => {
        setChildData(data?.childThreads ?? []);
    }, [data]);
    const addComment = useCallback((comment: Comment) => {
        // Make comment first, so you can see it without having to scroll to the bottom
        setChildData(curr => [{
            __typename: 'CommentThread',
            comment: comment as any,
            childThreads: [],
            endCursor: null,
            totalInThread: 0,
        }, ...curr]);
    }, []);
    const removeComment = useCallback((comment: Comment) => {
        setChildData(curr => [...curr.filter(c => c.comment.id !== comment.id)]);
    }, []);
    const updateComment = useCallback((comment: Comment) => {
        const index = childData.findIndex(c => c.comment.id === comment.id);
        if (index !== -1) return;
        setChildData(curr => updateArray(curr, index, {
            ...curr[index],
            comment,
        }));
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
                session={session}
                zIndex={zIndex}
            />;
        });
    }, [canOpen, childData, data, isOpen, language, session, zIndex]);

    return data && canOpen ? (
        <Stack direction="row" spacing={1} pl={2} pr={2}>
            {/* Comment connector */}
            <CommentConnector
                isOpen={isOpen}
                parentType={data.comment.creator?.__typename ?? 'User'}
                onToggle={() => setIsOpen(!isOpen)}
            />
            {/* Comment and child comments */}
            <Stack direction="column" spacing={1} sx={{ width: '100%' }}>
                {/* Comment */}
                <CommentThreadItem
                    data={data.comment}
                    handleCommentAdd={addComment}
                    handleCommentRemove={removeComment}
                    handleCommentUpdate={updateComment}
                    isOpen={isOpen}
                    language={language}
                    loading={false}
                    object={data.comment.commentedOn}
                    session={session}
                    zIndex={zIndex}
                />
                {/* Child comments */}
                {children}
                {/* "Show More" button */}
                {/* TODO */}
            </Stack>
        </Stack>
    ) : null
}