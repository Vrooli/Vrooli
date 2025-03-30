import { BookmarkFor, Comment, CommentFor, DeleteOneInput, DeleteType, ReactionFor, ReportFor, Success, endpointsActions, getTranslation, updateArray } from "@local/shared";
import { Avatar, Box, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SessionContext } from "../../../contexts.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { getYou, placeholderColor } from "../../../utils/display/listTools.js";
import { displayDate } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { ObjectType } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { CommentUpsert } from "../../../views/objects/comment/CommentUpsert.js";
import { BookmarkButton } from "../../buttons/BookmarkButton.js";
import { ReportButton } from "../../buttons/ReportButton/ReportButton.js";
import { ShareButton } from "../../buttons/ShareButton/ShareButton.js";
import { VoteButton } from "../../buttons/VoteButton.js";
import { TextLoading } from "../../lists/TextLoading/TextLoading.js";
import { OwnerLabel } from "../../text/OwnerLabel.js";
import { CommentConnectorProps, CommentThreadItemProps, CommentThreadProps } from "../../types.js";

/**
 * Collapsible, vertical line for indicating a comment level. Top of line 
 * if the profile image of the comment.
 */
export function CommentConnector({
    isOpen,
    parentType,
    onToggle,
}: CommentConnectorProps) {
    const { palette } = useTheme();

    // Random color for profile image (since we don't display custom image yet)
    const profileColors = useMemo(() => placeholderColor(), []);
    // Determine profile image type
    const profileIconInfo = useMemo(() => {
        switch (parentType) {
            case "Team":
                return { name: "Team", type: "Common" } as const;
            default:
                return { name: "User", type: "Common" } as const;
        }
    }, [parentType]);

    // Profile image
    const profileImage = useMemo(() => (
        <Avatar
            src="/broken-image.jpg" //TODO
            sx={{
                backgroundColor: profileColors[0],
                width: "48px",
                height: "48px",
                minWidth: "48px",
                minHeight: "48px",
            }}
        >
            <Icon
                fill={profileColors[1]}
                info={profileIconInfo}
            />
        </Avatar>
    ), [profileColors, profileIconInfo]);

    // If open, profile image on top of collapsible line
    if (isOpen) {
        return (
            <Stack direction="column">
                {/* Profile image */}
                {profileImage}
                {/* Collapsible, vertical line */}
                {
                    isOpen && <Box
                        width="5px"
                        height="100%"
                        borderRadius='100px'
                        bgcolor={profileColors[0]}
                        sx={{
                            marginLeft: "auto",
                            marginRight: "auto",
                            marginTop: 1,
                            marginBottom: 1,
                            cursor: "pointer",
                            "&:hover": {
                                brightness: palette.mode === "light" ? 1.05 : 0.95,
                            },
                        }}
                        onClick={onToggle}
                    />
                }
            </Stack>
        );
    }
    // If closed, OpenIcon to the left of profile image
    return (
        <Stack direction="row">
            <IconButton
                onClick={onToggle}
                sx={{
                    width: "48px",
                    height: "48px",
                }}
            >
                <IconCommon name="OpenThread" fill={profileColors[0]} />
            </IconButton>
            {profileImage}
        </Stack>
    );
}

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

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const handleDelete = useCallback(() => {
        if (!data) return;
        // Confirmation dialog
        PubSub.get().publish("alertDialog", {
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
                                <IconCommon name="Reply" fill="background.textSecondary" />
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
                                <IconCommon name="Delete" fill="background.textSecondary" />
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

/**
 * Comment and its list of child comments (which can have their own children and so on). 
 * Each level  contains a list of comment items, then a "Show more" text button.
 * To the left of this is a CommentConnector item, which is a collapsible line.
 */
export function CommentThread({
    canOpen,
    data,
    language,
}: CommentThreadProps) {
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
            />;
        });
    }, [canOpen, childData, data, isOpen, language]);

    return data && canOpen ? (
        <Stack direction="row" spacing={1} pl={2} pr={2}>
            {/* Comment connector */}
            <CommentConnector
                isOpen={isOpen}
                parentType={data.comment.owner?.__typename ?? "User"}
                onToggle={() => setIsOpen(!isOpen)}
            />
            {/* Comment and child comments */}
            <Stack direction="column" spacing={1} width="100%">
                {/* Comment */}
                <CommentThreadItem
                    data={data.comment}
                    handleCommentRemove={removeComment}
                    handleCommentUpsert={upsertComment}
                    isOpen={isOpen}
                    language={language}
                    loading={false}
                    object={data.comment.commentedOn}
                />
                {/* Child comments */}
                {children}
                {/* "Show More" button */}
                {/* TODO */}
            </Stack>
        </Stack>
    ) : null;
}
