import { Bookmark, BookmarkFor, uuidValidate } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { SelectBookmarkListDialog } from "components/dialogs/SelectBookmarkListDialog/SelectBookmarkListDialog";
import { SessionContext } from "contexts/SessionContext";
import { useBookmarker } from "hooks/useBookmarker";
import { BookmarkFilledIcon, BookmarkOutlineIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ActionCompletePayloads, ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { BookmarkButtonProps } from "../types";

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

    const [isSelectOpen, setIsSelectOpen] = useState(false);
    const openSelect = useCallback(() => { setIsSelectOpen(true); }, []);
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

    const handleClick = useCallback((event: any) => {
        if (!userId) return;
        const isBookmarked = !internalIsBookmarked;
        setInternalIsBookmarked(isBookmarked);
        // Prevent propagation of normal click event
        event.stopPropagation();
        event.preventDefault();
        // If objectId is not valid, return
        if (!uuidValidate(objectId)) return;
        // Call handleBookmark
        handleBookmark(isBookmarked);
    }, [objectId, internalIsBookmarked, userId, handleBookmark]);

    const Icon = internalIsBookmarked ? BookmarkFilledIcon : BookmarkOutlineIcon;
    const fill = useMemo<string>(() => {
        if (!userId || disabled) return "rgb(189 189 189)";
        if (internalIsBookmarked) return "#cbae30";
        return palette.secondary.main;
    }, [userId, disabled, internalIsBookmarked, palette]);

    return (
        <>
            {/* Dialog to select/deselect bookmark lists */}
            <SelectBookmarkListDialog
                objectId={objectId}
                objectType={bookmarkFor}
                onClose={(inList: boolean) => { closeSelect(); closeBookmarkDialog(); }}
                isCreate={!isBookmarkDialogOpen} // Hook only sets bookmark dialog when updating
                isOpen={isSelectOpen || isBookmarkDialogOpen}
            />
            {/* Main content */}
            <Tooltip title={t("Bookmark", { count: 1 })}>
                <IconButton
                    aria-label={t("Bookmark", { count: 1 })}
                    size="small"
                    onClick={handleClick}
                    sx={{
                        pointerEvents: disabled ? "none" : "auto",
                    }}
                >
                    <Icon fill={fill} />
                </IconButton>
            </Tooltip>
        </>
    );
}
