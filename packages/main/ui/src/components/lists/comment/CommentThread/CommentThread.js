import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { updateArray } from "../../../../utils/shape/general";
import { CommentConnector } from "../CommentConnector/CommentConnector";
import { CommentThreadItem } from "../CommentThreadItem/CommentThreadItem";
export const CommentThread = ({ canOpen, data, language, zIndex, }) => {
    const [isOpen, setIsOpen] = useState(true);
    const [childData, setChildData] = useState(data?.childThreads ?? []);
    useMemo(() => {
        setChildData(data?.childThreads ?? []);
    }, [data]);
    const removeComment = useCallback((comment) => {
        setChildData(curr => [...curr.filter(c => c.comment.id !== comment.id)]);
    }, []);
    const upsertComment = useCallback((comment) => {
        const index = childData.findIndex(c => c.comment.id === comment.id);
        if (index !== -1) {
            setChildData(curr => [{
                    __typename: "CommentThread",
                    comment: comment,
                    childThreads: [],
                    endCursor: null,
                    totalInThread: 0,
                }, ...curr]);
        }
        else {
            setChildData(curr => updateArray(curr, index, {
                ...curr[index],
                comment,
            }));
        }
    }, [childData]);
    const children = useMemo(() => {
        if (!data)
            return [];
        return childData.map((child, index) => {
            return _jsx(CommentThread, { canOpen: canOpen && isOpen, data: {
                    ...child,
                    childThreads: [],
                }, language: language, zIndex: zIndex }, `thread-${data.comment.id}-${index}`);
        });
    }, [canOpen, childData, data, isOpen, language, zIndex]);
    return data && canOpen ? (_jsxs(Stack, { direction: "row", spacing: 1, pl: 2, pr: 2, children: [_jsx(CommentConnector, { isOpen: isOpen, parentType: data.comment.owner?.__typename ?? "User", onToggle: () => setIsOpen(!isOpen) }), _jsxs(Stack, { direction: "column", spacing: 1, sx: { width: "100%" }, children: [_jsx(CommentThreadItem, { data: data.comment, handleCommentRemove: removeComment, handleCommentUpsert: upsertComment, isOpen: isOpen, language: language, loading: false, object: data.comment.commentedOn, zIndex: zIndex }), children] })] })) : null;
};
//# sourceMappingURL=CommentThread.js.map