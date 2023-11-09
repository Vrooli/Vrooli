import { BookmarkFor, Comment, CommentFor, DeleteOneInput, DeleteType, endpointPostDeleteOne, ReactionFor, ReportFor, Success } from "@local/shared";
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportButton } from "components/buttons/ReportButton/ReportButton";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { OwnerLabel } from "components/text/OwnerLabel/OwnerLabel";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { DeleteIcon, ReplyIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getYou } from "utils/display/listTools";
import { displayDate } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { ObjectType } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { CommentUpsert } from "views/objects/comment";
import { CommentThreadItemProps } from "../types";

export function CommentThreadItem({
    data,
    handleCommentRemove,
    handleCommentUpsert,
    isOpen,
    language,
    loading,
    object,
}: CommentThreadItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    const { objectId, objectType } = useMemo(() => ({
        objectId: object?.id,
        objectType: object?.__typename as CommentFor,
    }), [object]);
    const { isBookmarked, reaction } = useMemo(() => getYou(object as any), [object]);

    const { canDelete, canUpdate, canReply, canReport, canBookmark, canReact, displayText } = useMemo(() => {
        const { canDelete, canUpdate, canReply, canReport, canBookmark, canReact } = data?.you ?? {};
        const languages = getUserLanguages(session);
        const { text } = getTranslation(data, languages, true);
        return { canDelete, canUpdate, canReply, canReport, canBookmark, canReact, displayText: text };
    }, [data, session]);

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback(() => {
        if (!data) return;
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            messageKey: "DeleteConfirm",
            buttons: [
                {
                    labelKey: "Yes", onClick: () => {
                        fetchLazyWrapper<DeleteOneInput, Success>({
                            fetch: deleteMutation,
                            inputs: { id: data.id, objectType: DeleteType.Comment },
                            successCondition: (data) => data.success,
                            successMessage: () => ({ messageKey: "CommentDeleted" }),
                            onSuccess: () => {
                                handleCommentRemove(data);
                            },
                            errorMessage: () => ({ messageKey: "DeleteCommentFailed" }),
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [data, deleteMutation, handleCommentRemove]);

    const [isUpsertCommentOpen, setIsUpsertCommentOpen] = useState<boolean>(false);
    const [commentToUpdate, setCommentToUpdate] = useState<Comment | undefined>(undefined);
    const handleUpsertCommentOpen = useCallback((comment?: Comment) => {
        comment && setCommentToUpdate(comment);
        setIsUpsertCommentOpen(true);
    }, []);
    const handleUpsertCommentClose = useCallback(() => {
        setCommentToUpdate(undefined);
        setIsUpsertCommentOpen(false);
    }, []);
    const handleCommentCreated = useCallback((comment: Comment) => {
        handleUpsertCommentClose();
        handleCommentUpsert(comment);
    }, [handleCommentUpsert, handleUpsertCommentClose]);
    const handleCommentDeleted = useCallback(() => {
        handleUpsertCommentClose();
        if (!commentToUpdate) return;
        handleCommentRemove(commentToUpdate);
    }, [commentToUpdate, handleCommentRemove, handleUpsertCommentClose]);

    return (
        <>
            <ListItem
                id={`comment-${data?.id}`}
                disablePadding
                sx={{
                    display: "flex",
                    background: "transparent",
                }}
            >
                <Stack
                    direction="column"
                    spacing={1}
                    pl={2}
                    sx={{
                        width: "-webkit-fill-available",
                        display: "grid",
                    }}
                >
                    {/* Username and time posted */}
                    <Stack direction="row" spacing={1}>
                        {/* Username and role */}
                        {
                            <Stack direction="row" spacing={1} sx={{
                                overflow: "auto",
                            }}>
                                {objectType && <OwnerLabel
                                    objectType={objectType as unknown as ObjectType}
                                    owner={data?.owner}
                                    sxs={{
                                        label: {
                                            color: palette.background.textPrimary,
                                            fontWeight: "bold",
                                        },
                                    }} />}
                                {canUpdate && !(data?.owner?.id && data.owner.id === getCurrentUser(session).id) && <ListItemText
                                    primary={"(Can Edit)"}
                                    sx={{
                                        display: "flex",
                                        alignItems: "center",
                                        color: palette.mode === "light" ? "#fa4f4f" : "#f2a7a7",
                                    }}
                                />}
                                {data?.owner?.id && data.owner.id === getCurrentUser(session).id && <ListItemText
                                    primary={"(You)"}
                                    sx={{
                                        display: "flex",
                                        alignItems: "center",
                                        color: palette.mode === "light" ? "#fa4f4f" : "#f2a7a7",
                                    }}
                                />}
                            </Stack>
                        }
                        {/* Time posted */}
                        <ListItemText
                            primary={displayDate(data?.created_at, false)}
                            sx={{
                                display: "flex",
                                alignItems: "center",
                            }}
                        />
                    </Stack>
                    {/* Text */}
                    {isOpen && (loading ? <TextLoading /> : <ListItemText
                        primary={displayText}
                    />)}
                    {/* Text buttons for reply, share, report, star, delete. */}
                    {isOpen && <Stack direction="row" spacing={1}>
                        <VoteButton
                            direction="row"
                            disabled={!canReact}
                            emoji={reaction}
                            objectId={data?.id ?? ""}
                            voteFor={ReactionFor.Comment}
                            score={data?.score}
                            onChange={() => { }}
                        />
                        {canBookmark && <BookmarkButton
                            objectId={data?.id ?? ""}
                            bookmarkFor={BookmarkFor.Comment}
                            isBookmarked={isBookmarked ?? false}
                            showBookmarks={false}
                        />}
                        {canReply && <Tooltip title="Reply" placement='top'>
                            <IconButton
                                onClick={() => { handleUpsertCommentOpen(); }}
                            >
                                <ReplyIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </Tooltip>}
                        <ShareButton object={object} />
                        {canReport && <ReportButton
                            forId={data?.id ?? ""}
                            reportFor={objectType as any as ReportFor}
                        />}
                        {canDelete && <Tooltip title="Delete" placement='top'>
                            <IconButton
                                onClick={handleDelete}
                                disabled={loadingDelete}
                            >
                                <DeleteIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </Tooltip>}
                    </Stack>}
                    {/* Add/Update comment */}
                    {
                        objectId && objectType && <CommentUpsert
                            display="dialog"
                            overrideObject={commentToUpdate}
                            isCreate={!commentToUpdate}
                            isOpen={isUpsertCommentOpen}
                            language={language}
                            objectId={objectId}
                            objectType={objectType}
                            onCancel={handleUpsertCommentClose}
                            onClose={handleUpsertCommentClose}
                            onCompleted={handleCommentCreated}
                            onDeleted={handleCommentDeleted}
                            parent={(object as any) ?? null}
                        />
                    }
                </Stack>
            </ListItem>
        </>
    );
}
