import { jsx as _jsx } from "react/jsx-runtime";
import { BookmarkFilledIcon, BookmarkOutlineIcon } from "@local/icons";
import { uuidValidate } from "@local/uuid";
import { Box, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ObjectActionComplete } from "../../../utils/actions/objectActions";
import { getCurrentUser } from "../../../utils/authentication/session";
import { useBookmarker } from "../../../utils/hooks/useBookmarker";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const BookmarkButton = ({ disabled = false, isBookmarked = false, objectId, onChange, showBookmarks = true, bookmarkFor, bookmarks, sxs, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const [internalIsBookmarked, setInternalIsBookmarked] = useState(isBookmarked ?? null);
    useEffect(() => setInternalIsBookmarked(isBookmarked ?? false), [isBookmarked]);
    const onActionComplete = useCallback((action, data) => {
        console.log("action complete", action, data);
        switch (action) {
            case ObjectActionComplete.Bookmark:
                const listName = data.list.label;
                PubSub.get().publishSnack({
                    message: `Added to list "${listName}"`,
                    buttonKey: "Change",
                    buttonClicked: () => {
                        console.log("TODO");
                    },
                    severity: "Success",
                });
                break;
        }
    }, []);
    const { handleBookmark } = useBookmarker({
        objectId,
        objectType: bookmarkFor,
        onActionComplete,
    });
    const handleClick = useCallback((event) => {
        console.log("bookmark button click", objectId, internalIsBookmarked, userId, bookmarkFor);
        if (!userId)
            return;
        const isBookmarked = !internalIsBookmarked;
        setInternalIsBookmarked(isBookmarked);
        event.stopPropagation();
        event.preventDefault();
        if (!uuidValidate(objectId))
            return;
        handleBookmark(isBookmarked);
    }, [objectId, internalIsBookmarked, userId, bookmarkFor, handleBookmark]);
    const Icon = internalIsBookmarked ? BookmarkFilledIcon : BookmarkOutlineIcon;
    const fill = useMemo(() => {
        if (!userId || disabled)
            return "rgb(189 189 189)";
        if (internalIsBookmarked)
            return "#cbae30";
        return palette.secondary.main;
    }, [userId, disabled, internalIsBookmarked, palette]);
    return (_jsx(Box, { onClick: handleClick, sx: {
            marginRight: 0,
            marginTop: "auto !important",
            marginBottom: "auto !important",
            pointerEvents: disabled ? "none" : "all",
            cursor: userId ? "pointer" : "default",
            ...(sxs?.root ?? {}),
        }, children: _jsx(Icon, { fill: fill }) }));
};
//# sourceMappingURL=BookmarkButton.js.map