import { Bookmark, BookmarkFilledIcon, BookmarkFor, BookmarkOutlineIcon, uuidValidate } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { useBookmarker } from "utils/hooks/useBookmarker";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { BookmarkButtonProps } from "../types";

export const BookmarkButton = ({
    disabled = false,
    isBookmarked = false,
    objectId,
    onChange,
    showBookmarks = true,
    bookmarkFor,
    bookmarks,
    sxs,
}: BookmarkButtonProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsBookmarked, setInternalIsBookmarked] = useState<boolean | null>(isBookmarked ?? null);
    useEffect(() => setInternalIsBookmarked(isBookmarked ?? false), [isBookmarked]);

    const onActionComplete = useCallback((action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => {
        console.log("action complete", action, data);
        switch (action) {
            // When a bookmark is created, we assign a list automatically. 
            // So we must show a snackbar to inform the user that the bookmark was created, 
            // with an option to change the list.
            case ObjectActionComplete.Bookmark:
                const listName = (data as Bookmark).list.label;
                PubSub.get().publishSnack({
                    message: `Added to list "${listName}"`,
                    buttonKey: "Change",
                    buttonClicked: () => {
                        console.log("TODO");
                    },
                    severity: "Success",
                });
                break;
            // When bookmark is removed, we don't need to do anything
        }
    }, []);

    const { handleBookmark } = useBookmarker({
        objectId,
        objectType: bookmarkFor as BookmarkFor,
        onActionComplete,
    });

    const handleClick = useCallback((event: any) => {
        console.log("bookmark button click", objectId, internalIsBookmarked, userId, bookmarkFor);
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
    }, [objectId, internalIsBookmarked, userId, bookmarkFor, handleBookmark]);

    const Icon = internalIsBookmarked ? BookmarkFilledIcon : BookmarkOutlineIcon;
    const fill = useMemo<string>(() => {
        if (!userId || disabled) return "rgb(189 189 189)";
        if (internalIsBookmarked) return "#cbae30";
        return palette.secondary.main;
    }, [userId, disabled, internalIsBookmarked, palette]);

    return (
        <Box
            onClick={handleClick}
            sx={{
                marginRight: 0,
                marginTop: "auto !important",
                marginBottom: "auto !important",
                pointerEvents: disabled ? "none" : "all",
                cursor: userId ? "pointer" : "default",
                ...(sxs?.root ?? {}),
            }}
        >
            <Icon fill={fill} />
        </Box>
    );
};
