import { Box, ListItemText, Stack, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { BookmarkButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { uuidValidate } from '@shared/uuid';
import { BookmarkFilledIcon, BookmarkOutlineIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { PubSub } from 'utils';
import { documentNodeWrapper } from 'api/utils';

export const BookmarkButton = ({
    disabled = false,
    isBookmarked = false,
    objectId,
    onChange,
    session,
    showBookmarks = true,
    bookmarkFor,
    bookmarks,
    sxs,
}: BookmarkButtonProps) => {
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsBookmarked, setInternalIsBookmarked] = useState<boolean | null>(isBookmarked ?? null);
    useEffect(() => setInternalIsBookmarked(isBookmarked ?? false), [isBookmarked]);

    const internalBookmarks: number | null = useMemo(() => {
        if (bookmarks === null || bookmarks === undefined) return null;
        const count = bookmarks;
        if (internalIsBookmarked === true && isBookmarked === false) return count + 1;
        if (internalIsBookmarked === false && isBookmarked === true) return count - 1;
        return count;
    }, [internalIsBookmarked, isBookmarked, bookmarks]);

    const handleClick = useCallback((event: any) => {
        if (!userId) return;
        const isBookmarked = !internalIsBookmarked;
        setInternalIsBookmarked(isBookmarked);
        // Prevent propagation of normal click event
        event.stopPropagation();
        // If objectId is not valid, return
        if (!uuidValidate(objectId)) return;
        // If not isBookmarked, add to default bookmark
        //TODO
        // Show snack message that bookmark was added, with option to set label
        //TODO
        // Else if isBookmarked, query for bookmarks on this object
        //TODO
        // If there is only one bookmark, delete it
        //TODO
        // If there are multiple, open dialog to select which bookmark to delete
        //TODO
        // Send star mutation
        // documentNodeWrapper<Success, StarInput>({
        //     node: starStar,
        //     input: { isBookmarked, bookmarkFor, forConnect: objectId },
        //     onSuccess: () => { 
        //         if (onChange) onChange(isBookmarked, event) 
        //         PubSub.get().publishSnack({ messageKey: isBookmarked ? 'FavoritesAdded' : 'FavoritesRemoved', severity: 'Success' });
        //     },
        // })
    }, [userId, internalIsBookmarked, bookmarkFor, objectId, onChange]);

    const Icon = internalIsBookmarked ? BookmarkFilledIcon : BookmarkOutlineIcon;
    const fill = useMemo<string>(() => {
        if (!userId || disabled) return 'rgb(189 189 189)';
        if (internalIsBookmarked) return '#cbae30';
        return palette.secondary.main;
    }, [userId, disabled, internalIsBookmarked, palette]);

    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
                marginTop: 'auto !important',
                marginBottom: 'auto !important',
                pointerEvents: 'none',
                ...(sxs?.root ?? {}),
            }}
        >
            <Box onClick={handleClick} sx={{
                display: 'contents',
                cursor: userId ? 'pointer' : 'default',
                pointerEvents: disabled ? 'none' : 'all',
            }}>
                <Icon fill={fill} />
            </Box>
            {showBookmarks && internalBookmarks !== null && <ListItemText
                primary={internalBookmarks}
                sx={{ ...multiLineEllipsis(1), pointerEvents: 'none' }}
            />}
        </Stack>
    )
}