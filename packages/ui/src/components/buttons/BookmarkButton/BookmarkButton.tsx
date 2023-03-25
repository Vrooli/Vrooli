import { Box, useTheme } from '@mui/material';
import { BookmarkFor } from '@shared/consts';
import { BookmarkFilledIcon, BookmarkOutlineIcon } from '@shared/icons';
import { uuidValidate } from '@shared/uuid';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { getCurrentUser } from 'utils/authentication/session';
import { useBookmarker } from 'utils/hooks/useBookmarker';
import { SessionContext } from 'utils/SessionContext';
import { BookmarkButtonProps } from '../types';

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

    const { handleBookmark } = useBookmarker({
        objectId,
        objectType: bookmarkFor as BookmarkFor,
        onActionComplete: () => { },//TODO
    });

    const handleClick = useCallback((event: any) => {
        console.log('bookmark button click', objectId, internalIsBookmarked, userId, bookmarkFor);
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
        if (!userId || disabled) return 'rgb(189 189 189)';
        if (internalIsBookmarked) return '#cbae30';
        return palette.secondary.main;
    }, [userId, disabled, internalIsBookmarked, palette]);

    return (
        <Box
            onClick={handleClick}
            sx={{
                marginRight: 0,
                marginTop: 'auto !important',
                marginBottom: 'auto !important',
                pointerEvents: disabled ? 'none' : 'all',
                cursor: userId ? 'pointer' : 'default',
                ...(sxs?.root ?? {}),
            }}
        >
            <Icon fill={fill} />
        </Box>
    )
}