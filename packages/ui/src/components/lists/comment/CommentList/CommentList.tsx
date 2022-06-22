import { Stack } from "@mui/material";
import { useMemo, useState } from "react";
import { CommentConnector } from "../CommentConnector/CommentConnector";
import { CommentListItem } from "../CommentListItem/CommentListItem";
import { CommentListProps } from "../types";

/**
 * Comment and its list of child comments (which can have their own children and so on). 
 * Each level  contains a list of comment items, then a "Show more" text button.
 * To the left of this is a CommentConnector item, which is a collapsible line.
 */
export const CommentList = ({
    data,
    session
}: CommentListProps) => {
    // open state
    const [isOpen, setIsOpen] = useState(true);
    // list of comment items
    const children = useMemo(() => {
        if (!data) return [];
        return data.childThreads.map((child, index) => {
            return <CommentList
                key={`thread-${data.comment.id}-${index}`}
                data={{
                    ...child,
                    childThreads: [],
                }}
                session={session}
            />;
        });
    }, [data, session]);

    return data ? (
        <Stack direction="row" spacing={1} pl={2} pr={2}>
            {/* Comment connector */}
            <CommentConnector
                isOpen={isOpen}
                objectType={data.comment.commentedOn.__typename}
                onToggle={() => setIsOpen(!isOpen)}
            />
            {/* Comment and child comments */}
            <Stack direction="column" spacing={1}>
                {/* Comment */}
                <CommentListItem
                    data={data.comment}
                    loading={false}
                    session={session}
                />
                {/* Child comments */}
                {children}
            </Stack>
        </Stack>
    ) : null
}