import { Box, ListItemText, Stack, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StarButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { uuidValidate } from '@shared/uuid';
import { StarFilledIcon, StarOutlineIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { PubSub } from 'utils';
import { SnackSeverity } from 'components/dialogs';
import { documentNodeWrapper } from 'api/utils';
import { StarInput, Success } from '@shared/consts';
import { starStar } from 'api/generated/endpoints/star';

export const BookmarkButton = ({
    disabled = false,
    isStar = false,
    objectId,
    onChange,
    session,
    showBookmarks = true,
    starFor,
    bookmarks,
    sxs,
    tooltipPlacement = "left"
}: StarButtonProps) => {
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsStar, setInternalIsStar] = useState<boolean | null>(isStar ?? null);
    useEffect(() => setInternalIsStar(isStar ?? false), [isStar]);

    const internalBookmarks: number | null = useMemo(() => {
        if (bookmarks === null || bookmarks === undefined) return null;
        const starNum = bookmarks;
        if (internalIsStar === true && isStar === false) return starNum + 1;
        if (internalIsStar === false && isStar === true) return starNum - 1;
        return starNum;
    }, [internalIsStar, isStar, bookmarks]);

    const handleClick = useCallback((event: any) => {
        console.log('starbutton handleclick', userId, event, internalIsStar, starFor, objectId, onChange)
        if (!userId) return;
        const isStar = !internalIsStar;
        setInternalIsStar(isStar);
        // Prevent propagation of normal click event
        event.stopPropagation();
        // If objectId is not valid, return
        if (!uuidValidate(objectId)) return;
        // Send star mutation
        documentNodeWrapper<Success, StarInput>({
            node: starStar,
            input: { isStar, starFor, forConnect: objectId },
            onSuccess: () => { 
                if (onChange) onChange(isStar, event) 
                PubSub.get().publishSnack({ messageKey: isStar ? 'FavoritesAdded' : 'FavoritesRemoved', severity: SnackSeverity.Success });
            },
        })
    }, [userId, internalIsStar, starFor, objectId, onChange]);

    const Icon = internalIsStar ? StarFilledIcon : StarOutlineIcon;
    const fill = useMemo<string>(() => {
        if (!userId || disabled) return 'rgb(189 189 189)';
        if (internalIsStar) return '#cbae30';
        return palette.secondary.main;
    }, [userId, disabled, internalIsStar, palette]);

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