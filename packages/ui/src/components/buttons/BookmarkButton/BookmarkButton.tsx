import { Bookmark, BookmarkFor, uuidValidate } from "@local/shared";
import { Box, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { SelectBookmarkListDialog } from "components/dialogs/SelectBookmarkListDialog/SelectBookmarkListDialog";
import { SessionContext } from "contexts";
import { useBookmarkListsStore, useBookmarker } from "hooks/objectActions";
import { BookmarkFilledIcon, BookmarkOutlineIcon } from "icons";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ActionCompletePayloads, ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { BookmarkButtonProps } from "../types";

const BookmarksLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    marginLeft: theme.spacing(0.5),
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
        if (!uuidValidate(objectId)) return;
        // Call handleBookmark
        handleBookmark(isBookmarked);
    }, [userId, internalIsBookmarked, onChange, objectId, handleBookmark]);

    const handleCloseList = useCallback(function handleCloseListCallback() {
        closeSelect();
        closeBookmarkDialog();
    }, [closeSelect, closeBookmarkDialog]);

    const Icon = internalIsBookmarked ? BookmarkFilledIcon : BookmarkOutlineIcon;
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
                    component="button"
                    aria-label={t("Bookmark", { count: 1 })}
                    onClick={handleClick}
                    sx={bookmarkButtonStyle}
                >
                    <Icon fill={fill} />
                    {showBookmarks && typeof bookmarks === "number" && (
                        <BookmarksLabel>
                            {bookmarks}
                        </BookmarksLabel>
                    )}
                </Box>
            </Tooltip>
        </>
    );
}
