import { Box, ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StarButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { documentNodeWrapper } from 'graphql/utils/graphqlWrapper';
import { starVariables, star_star } from 'graphql/generated/star';
import { starMutation } from 'graphql/mutation';
import { uuidValidate } from '@shared/uuid';
import { StarFilledIcon, StarOutlineIcon } from '@shared/icons';

export const StarButton = ({
    isStar = false,
    objectId,
    onChange,
    session,
    showStars = true,
    starFor,
    stars,
    sxs,
    tooltipPlacement = "left"
}: StarButtonProps) => {
    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsStar, setInternalIsStar] = useState<boolean | null>(isStar ?? null);
    useEffect(() => setInternalIsStar(isStar ?? false), [isStar]);

    const internalStars: number | null = useMemo(() => {
        if (stars === null || stars === undefined) return null;
        const starNum = stars;
        if (internalIsStar === true && isStar === false) return starNum + 1;
        if (internalIsStar === false && isStar === true) return starNum - 1;
        return starNum;
    }, [internalIsStar, isStar, stars]);

    const handleClick = useCallback((event: any) => {
        if (!session.id) return;
        const isStar = !internalIsStar;
        setInternalIsStar(isStar);
        // Prevent propagation of normal click event
        event.stopPropagation();
        // If objectId is not valid, return
        if (!uuidValidate(objectId)) return;
        // Send star mutation
        documentNodeWrapper<star_star, starVariables>({
            node: starMutation,
            input: { isStar, starFor, forId: objectId },
            onSuccess: () => { if (onChange) onChange(isStar, event) },
        })
    }, [session.id, internalIsStar, starFor, objectId, onChange]);

    const Icon = internalIsStar ? StarFilledIcon : StarOutlineIcon;
    const tooltip = internalIsStar ? 'Remove from favorites' : 'Add to favorites';
    const color = session?.id ? '#cbae30' : 'rgb(189 189 189)';
    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
                marginTop: 'auto !important',
                marginBottom: 'auto !important',
                ...(sxs?.root ?? {}),
            }}
        >
            <Tooltip placement={tooltipPlacement} title={tooltip}>
                <Box onClick={handleClick} sx={{ display: 'contents', cursor: session?.id ? 'pointer' : 'default' }}>
                    <Icon fill={color} />
                </Box>
            </Tooltip>
            {showStars && internalStars !== null && <ListItemText
                primary={internalStars}
                sx={{ ...multiLineEllipsis(1) }}
            />}
        </Stack>
    )
}