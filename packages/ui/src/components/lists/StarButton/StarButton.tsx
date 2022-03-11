import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StarButtonProps } from '../types';
import {
    Star as IsStarredIcon,
    StarBorder as IsNotStarredIcon,
} from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useMutation } from '@apollo/client';
import { star } from 'graphql/generated/star';
import { starMutation } from 'graphql/mutation';

export const StarButton = ({
    session,
    isStar = false,
    stars = 0,
    objectId,
    showStars = true,
    starFor,
    onChange,
    tooltipPlacement = "left"
}: StarButtonProps) => {
    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsStar, setInternalIsStar] = useState<boolean | null>(isStar ?? null);
    useEffect(() => setInternalIsStar(isStar ?? false), [isStar]);

    const internalStars = useMemo(() => {
        const starNum = stars ?? 0;
        if (internalIsStar === true && isStar === false) return starNum + 1;
        if (internalIsStar === false && isStar === true) return starNum - 1;
        return starNum;
    }, [internalIsStar, isStar, stars]);

    const [mutation] = useMutation<star>(starMutation);
    const handleClick = useCallback((event: any) => {
        if (!session.id) return;
        const isStar = !internalIsStar;
        setInternalIsStar(isStar);
        // Prevent propagation of normal click event
        event.stopPropagation();
        // Send star mutation
        mutationWrapper({
            mutation,
            input: { isStar, starFor, forId: objectId },
            onSuccess: (response) => { console.log('STAR SUCCESS', response.data.star); onChange(response.data.star) },
        })
    }, [session.id, internalIsStar, objectId, starFor]);

    const Icon = internalIsStar ? IsStarredIcon : IsNotStarredIcon;
    const tooltip = internalIsStar ? 'Remove from favorites' : 'Add to favorites';
    const color = session?.id ? '#cbae30' : 'rgb(189 189 189)';
    return (
        <Stack
            direction="row"
            spacing={1}
            sx={{
                marginRight: 0,
            }}
        >
            <Tooltip placement={tooltipPlacement} title={tooltip}>
                <Icon onClick={handleClick} sx={{ fill: color, cursor: session?.id ? 'pointer' : 'default' }} />
            </Tooltip>
            { showStars ? <ListItemText
                primary={internalStars}
                sx={{ ...multiLineEllipsis(1) }}
            /> : null }
        </Stack>
    )
}