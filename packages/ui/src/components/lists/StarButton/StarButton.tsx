import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StarButtonProps } from '../types';
import {
    Star as IsStarredIcon,
    StarBorder as IsNotStarredIcon,
} from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { useMutation } from '@apollo/client';
import { star, starVariables } from 'graphql/generated/star';
import { starMutation } from 'graphql/mutation';
import { validate as uuidValidate } from 'uuid';

export const StarButton = ({
    session,
    isStar = false,
    stars,
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

    const internalStars: number | null = useMemo(() => {
        if (!stars) return null;
        const starNum = stars;
        if (internalIsStar === true && isStar === false) return starNum + 1;
        if (internalIsStar === false && isStar === true) return starNum - 1;
        return starNum;
    }, [internalIsStar, isStar, stars]);

    const [mutation] = useMutation<star, starVariables>(starMutation);
    const handleClick = useCallback((event: any) => {
        if (!session.id) return;
        const isStar = !internalIsStar;
        setInternalIsStar(isStar);
        // Prevent propagation of normal click event
        event.stopPropagation();
        // If objectId is not valid, return
        if (!uuidValidate(objectId)) return;
        // Send star mutation
        mutationWrapper({
            mutation,
            input: { isStar, starFor, forId: objectId },
            onSuccess: () => { if (onChange) onChange(isStar) },
        })
    }, [session.id, internalIsStar, mutation, starFor, objectId, onChange]);

    const Icon = internalIsStar ? IsStarredIcon : IsNotStarredIcon;
    const tooltip = internalIsStar ? 'Remove from favorites' : 'Add to favorites';
    const color = session?.id ? '#cbae30' : 'rgb(189 189 189)';
    return (
        <Stack
            direction="row"
            spacing={1}
            sx={{
                marginRight: 0,
                marginTop: 'auto !important',
                marginBottom: 'auto !important',
            }}
        >
            <Tooltip placement={tooltipPlacement} title={tooltip}>
                <Icon onClick={handleClick} sx={{ fill: color, cursor: session?.id ? 'pointer' : 'default' }} />
            </Tooltip>
            { showStars && stars ? <ListItemText
                primary={internalStars}
                sx={{ ...multiLineEllipsis(1) }}
            /> : null }
        </Stack>
    )
}