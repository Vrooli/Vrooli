import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BookmarkFor, DeleteType, ReactionFor } from "@local/consts";
import { DeleteIcon, ReplyIcon } from "@local/icons";
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { deleteOneOrManyDeleteOne } from "../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCustomMutation } from "../../../../api/hooks";
import { mutationWrapper } from "../../../../api/utils";
import { getCurrentUser } from "../../../../utils/authentication/session";
import { getYou } from "../../../../utils/display/listTools";
import { displayDate } from "../../../../utils/display/stringTools";
import { getTranslation, getUserLanguages } from "../../../../utils/display/translationTools";
import { PubSub } from "../../../../utils/pubsub";
import { SessionContext } from "../../../../utils/SessionContext";
import { BookmarkButton } from "../../../buttons/BookmarkButton/BookmarkButton";
import { ReportButton } from "../../../buttons/ReportButton/ReportButton";
import { ShareButton } from "../../../buttons/ShareButton/ShareButton";
import { VoteButton } from "../../../buttons/VoteButton/VoteButton";
import { CommentUpsertInput } from "../../../inputs/CommentUpsertInput/CommentUpsertInput";
import { OwnerLabel } from "../../../text/OwnerLabel/OwnerLabel";
import { TextLoading } from "../../TextLoading/TextLoading";
export function CommentThreadItem({ data, handleCommentRemove, handleCommentUpsert, isOpen, language, loading, object, zIndex, }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { objectId, objectType } = useMemo(() => ({
        objectId: object?.id,
        objectType: object?.__typename,
    }), [object]);
    const { isBookmarked, reaction } = useMemo(() => getYou(object), [object]);
    const { canDelete, canUpdate, canReply, canReport, canBookmark, canReact, displayText } = useMemo(() => {
        const { canDelete, canUpdate, canReply, canReport, canBookmark, canReact } = data?.you ?? {};
        const languages = getUserLanguages(session);
        const { text } = getTranslation(data, languages, true);
        return { canDelete, canUpdate, canReply, canReport, canBookmark, canReact, displayText: text };
    }, [data, session]);
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback(() => {
        if (!data)
            return;
        PubSub.get().publishAlertDialog({
            messageKey: "DeleteCommentConfirm",
            buttons: [
                {
                    labelKey: "Yes", onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: data.id, objectType: DeleteType.Comment },
                            successCondition: (data) => data.success,
                            successMessage: () => ({ key: "CommentDeleted" }),
                            onSuccess: () => {
                                handleCommentRemove(data);
                            },
                            errorMessage: () => ({ key: "DeleteCommentFailed" }),
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [data, deleteMutation, handleCommentRemove]);
    const [isUpsertCommentOpen, setIsUpsertCommentOpen] = useState(false);
    const [commentToUpdate, setCommentToUpdate] = useState(undefined);
    const handleUpsertCommentOpen = useCallback((comment) => {
        comment && setCommentToUpdate(comment);
        setIsUpsertCommentOpen(true);
    }, []);
    const handleUpsertCommentClose = useCallback(() => {
        setCommentToUpdate(undefined);
        setIsUpsertCommentOpen(false);
    }, []);
    return (_jsx(_Fragment, { children: _jsx(ListItem, { id: `comment-${data?.id}`, disablePadding: true, sx: {
                display: "flex",
                background: "transparent",
            }, children: _jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: {
                    width: "-webkit-fill-available",
                    display: "grid",
                }, children: [_jsxs(Stack, { direction: "row", spacing: 1, children: [_jsxs(Stack, { direction: "row", spacing: 1, sx: {
                                    overflow: "auto",
                                }, children: [objectType && _jsx(OwnerLabel, { objectType: objectType, owner: data?.owner, sxs: {
                                            label: {
                                                color: palette.background.textPrimary,
                                                fontWeight: "bold",
                                            },
                                        } }), canUpdate && !(data?.owner?.id && data.owner.id === getCurrentUser(session).id) && _jsx(ListItemText, { primary: "(Can Edit)", sx: {
                                            display: "flex",
                                            alignItems: "center",
                                            color: palette.mode === "light" ? "#fa4f4f" : "#f2a7a7",
                                        } }), data?.owner?.id && data.owner.id === getCurrentUser(session).id && _jsx(ListItemText, { primary: "(You)", sx: {
                                            display: "flex",
                                            alignItems: "center",
                                            color: palette.mode === "light" ? "#fa4f4f" : "#f2a7a7",
                                        } })] }), _jsx(ListItemText, { primary: displayDate(data?.created_at, false), sx: {
                                    display: "flex",
                                    alignItems: "center",
                                } })] }), isOpen && (loading ? _jsx(TextLoading, {}) : _jsx(ListItemText, { primary: displayText })), isOpen && _jsxs(Stack, { direction: "row", spacing: 1, children: [_jsx(VoteButton, { direction: "row", disabled: !canReact, emoji: reaction, objectId: data?.id ?? "", voteFor: ReactionFor.Comment, score: data?.score, onChange: () => { } }), canBookmark && _jsx(BookmarkButton, { objectId: data?.id ?? "", bookmarkFor: BookmarkFor.Comment, isBookmarked: isBookmarked ?? false, showBookmarks: false }), canReply && _jsx(Tooltip, { title: "Reply", placement: 'top', children: _jsx(IconButton, { onClick: () => { handleUpsertCommentOpen(); }, children: _jsx(ReplyIcon, { fill: palette.background.textSecondary }) }) }), _jsx(ShareButton, { object: object, zIndex: zIndex }), canReport && _jsx(ReportButton, { forId: data?.id ?? "", reportFor: objectType, zIndex: zIndex }), canDelete && _jsx(Tooltip, { title: "Delete", placement: 'top', children: _jsx(IconButton, { onClick: handleDelete, disabled: loadingDelete, children: _jsx(DeleteIcon, { fill: palette.background.textSecondary }) }) })] }), isUpsertCommentOpen && objectId && objectType && _jsx(CommentUpsertInput, { comment: commentToUpdate, language: language, objectId: objectId, objectType: objectType, onCancel: handleUpsertCommentClose, onCompleted: handleCommentUpsert, parent: object ?? null, zIndex: zIndex })] }) }) }));
}
//# sourceMappingURL=CommentThreadItem.js.map