import { Bookmark, BookmarkFor, validatePK } from "@local/shared";
import { Box, Tooltip, Typography, styled, useTheme } from "@mui/material";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useBookmarker } from "../../hooks/objectActions.js";
import { IconCommon } from "../../icons/Icons.js";
import { useBookmarkListsStore } from "../../stores/bookmarkListsStore.js";
import { ActionCompletePayloads, ObjectActionComplete } from "../../utils/actions/objectActions.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { PubSub } from "../../utils/pubsub.js";
import { SelectBookmarkListDialog } from "../dialogs/SelectBookmarkListDialog/SelectBookmarkListDialog.js";
import { BookmarkButtonProps } from "./types.js";

const BookmarksLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    marginLeft: theme.spacing(0.5),
    verticalAlign: "text-bottom",
}));

export function BookmarkButton({
    disabled = false,
    isBookmarked = false,
    objectId,
    onChange,
    showBookmarks = true,
    bookmarkFor,
    bookmarks,
    sxs,
}: BookmarkButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsBookmarked, setInternalIsBookmarked] = useState<boolean | null>(isBookmarked ?? null);
    useEffect(() => setInternalIsBookmarked(isBookmarked ?? false), [isBookmarked]);

    const bookmarkLists = useBookmarkListsStore(state => state.bookmarkLists);
    const fetchBookmarkLists = useBookmarkListsStore(state => state.fetchBookmarkLists);
    const isBookmarkListsLoading = useBookmarkListsStore(state => state.isLoading);

    const [isSelectOpen, setIsSelectOpen] = useState(false);
    const openSelect = useCallback(async function openSelectCallback() {
        // If the bookmark lists are not loaded yet, fetch them
        if (bookmarkLists.length === 0 && !isBookmarkListsLoading) {
            await fetchBookmarkLists();
        }
        setIsSelectOpen(true);
    }, [bookmarkLists.length, isBookmarkListsLoading, fetchBookmarkLists]);
    const closeSelect = useCallback(() => { setIsSelectOpen(false); }, []);

    const onActionComplete = useCallback(<T extends keyof ActionCompletePayloads>(action: T, data: ActionCompletePayloads[T]) => {
        switch (action) {
            // When a bookmark is created, we assign a list automatically. 
            // So we must show a snackbar to inform the user that the bookmark was created, 
            // with an option to change the list.
            case ObjectActionComplete.Bookmark: {
                const listName = (data as Bookmark).list.label;
                PubSub.get().publish("snack", {
                    message: `Added to list "${listName}"`,
                    buttonKey: "Change",
                    buttonClicked: () => {
                        // Open the select dialog
                        openSelect();
                    },
                    severity: "Success",
                });
                break;
            }
            // When bookmark is removed, we don't need to do anything
        }
    }, [openSelect]);

    const {
        isBookmarkDialogOpen,
        handleBookmark,
        closeBookmarkDialog,
    } = useBookmarker({
        objectId,
        objectType: bookmarkFor as BookmarkFor,
        onActionComplete,
    });

    const handleClick = useCallback(function handleClickCallback(event: React.MouseEvent<HTMLButtonElement>) {
        if (!userId) return;
        const isBookmarked = !internalIsBookmarked;
        setInternalIsBookmarked(isBookmarked);
        onChange?.(isBookmarked);
        // Prevent propagation of normal click event
        event.stopPropagation();
        event.preventDefault();
        // If objectId is not valid, return
        if (!validatePK(objectId)) return;
        // Call handleBookmark
        handleBookmark(isBookmarked);
    }, [userId, internalIsBookmarked, onChange, objectId, handleBookmark]);

    const handleCloseList = useCallback(function handleCloseListCallback() {
        closeSelect();
        closeBookmarkDialog();
    }, [closeSelect, closeBookmarkDialog]);

    const iconType = internalIsBookmarked ? "BookmarkFilled" as const : "BookmarkOutline" as const;
    const ariaLabel = internalIsBookmarked ? t("BookmarkUndo", { count: 1 }) : t("AddBookmark");
    const fill = useMemo<string>(() => {
        if (userId && !disabled && internalIsBookmarked) return "#cbae30";
        return palette.background.textSecondary;
    }, [userId, disabled, internalIsBookmarked, palette]);

    const bookmarkButtonStyle = useMemo(function bookmarkButtonStyleMemo() {
        return {
            display: "flex",
            flexDirection: "row",
            border: "none",
            background: "transparent",
            cursor: disabled ? "not-allowed" : "pointer",
            pointerEvents: disabled ? "none" : "auto",
            alignItems: "center",
            padding: 0,
            ...sxs?.root,
        } as const;
    }, [disabled, sxs?.root]);

    return (
        <>
            <SelectBookmarkListDialog
                objectId={objectId}
                objectType={bookmarkFor}
                onClose={handleCloseList}
                isCreate={!isBookmarkDialogOpen} // Hook only sets bookmark dialog when updating
                isOpen={isSelectOpen || isBookmarkDialogOpen}
            />
            {/* Main content */}
            <Tooltip title={t("Bookmark", { count: 1 })}>
                <Box
                    aria-label={ariaLabel}
                    aria-pressed={internalIsBookmarked === true}
                    component="button"
                    onClick={handleClick}
                    sx={{
                        ...bookmarkButtonStyle,
                        display: "inline-flex",
                        alignItems: "center",
                    }}
                >
                    <Box sx={{ display: "flex", alignItems: "center" }}>
                        <IconCommon
                            decorative
                            fill={fill}
                            name={iconType}
                            size={24}
                        />
                        {showBookmarks && typeof bookmarks === "number" && (
                            <BookmarksLabel
                                variant="body2"
                                sx={{
                                    lineHeight: 1,
                                    paddingTop: "2px"
                                }}
                            >
                                {bookmarks}
                            </BookmarksLabel>
                        )}
                    </Box>
                </Box>
            </Tooltip>
        </>
    );
}
