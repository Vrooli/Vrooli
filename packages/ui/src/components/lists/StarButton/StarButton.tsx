import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StarButtonProps } from '../types';
import {
    Star as IsStarredIcon,
    StarBorder as IsNotStarredIcon,
} from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';

export const StarButton = ({
    session,
    isStar = false,
    stars = 0,
    onStar,
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

    const handleClick = useCallback((event: any) => {
        if (!session.id) return;
        const isStar = !internalIsStar;
        setInternalIsStar(isStar);
        onStar(event, isStar);
    }, [internalIsStar, session.id, onStar]);

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
            <ListItemText
                primary={internalStars}
                sx={{ ...multiLineEllipsis(1) }}
            />
        </Stack>
    )
}