import {
    Box,
    IconButton,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import { openLink } from 'utils';
import { useCallback, useEffect, useMemo } from 'react';
import { readOpenGraphQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { useLocation } from 'wouter';
import { ResourceCardProps } from '../types';
import { cardRoot } from '../styles';
import { multiLineEllipsis } from 'styles';
import { readOpenGraph, readOpenGraphVariables } from 'graphql/generated/readOpenGraph';
import {
    Delete as DeleteIcon,
    Edit as EditIcon,
} from '@mui/icons-material';

const buttonProps = {
    position: 'absolute',
    background: 'white',
    top: '-15px',
    color: (t) => t.palette.secondary.dark,
    borderRadius: '100%',
    transition: 'brightness 0.2s ease-in-out',
    '&:hover': {
        filter: `brightness(120%)`,
        background: 'white',
    },
}

export const ResourceCard = ({
    data,
    Icon,
    onClick,
    onRightClick,
}: ResourceCardProps) => {
    const [, setLocation] = useLocation();
    const [getOpenGraphData, { data: queryResult }] = useLazyQuery<readOpenGraph, readOpenGraphVariables>(readOpenGraphQuery);
    const queryData = useMemo(() => queryResult?.readOpenGraph, [queryResult]);
    const title = useMemo(() => data?.title ?? queryData?.title, [data, queryData]);

    const handleClick = useCallback((event: any) => {
        if (onClick) onClick(data);
        else if (data.link) openLink(setLocation, data.link);
    }, [onClick, data]);
    const handleRightClick = useCallback((event: any) => {
        if (onRightClick) onRightClick(event, data);
    }, [onRightClick, data]);

    useEffect(() => {
        if (data.link && !data.title) {
            getOpenGraphData({ variables: { input: { url: data.link } } })
        }
    }, [getOpenGraphData, data])

    // const display = useMemo(() => {
    //     if (!data || !data.displayUrl) {
    //         return <NoImageWithTextIcon style={{ aspectRatio: '1' }} />
    //     }
    //     return (
    //         <CardMedia
    //             component="img"
    //             src={data.displayUrl}
    //             sx={{ aspectRatio: '1' }}
    //             alt={`Image from ${data.link ?? data.title}`}
    //             title={data.title ?? data.link}
    //         />
    //     )
    // }, [data])

    return (
        <Tooltip placement="top" title={data?.description ?? queryData?.description ?? ''}>
            <Box
                onClick={handleClick}
                onContextMenu={handleRightClick}
                sx={{
                    ...cardRoot,
                    padding: 1,
                    width: '120px',
                    minWidth: '120px',
                    position: 'relative'
                }}
            >
                {/* Delete/edit buttons */}
                <Box sx={{
                    width: '100%',
                    height: '100%',
                    position: 'absolute',
                    top: '0',
                    left: '0',
                    opacity: '0',
                    transition: 'opacity 0.2s ease-in-out',
                    '&:hover': {
                        opacity: '1',
                    },
                }}>
                    <IconButton size="small" onClick={() => { }} aria-label="close" sx={{ ...buttonProps, left: '-15px' } as any}>
                        <DeleteIcon />
                    </IconButton>
                    <IconButton size="small" onClick={() => { }} aria-label="close" sx={{ ...buttonProps, right: '-15px' } as any}>
                        <EditIcon />
                    </IconButton>
                </Box>
                {/* Content */}
                <Stack direction="column" justifyContent="center" alignItems="center">
                    <Icon sx={{ fill: 'white' }} />
                    <Typography
                        gutterBottom
                        variant="body2"
                        component="h3"
                        sx={{ ...multiLineEllipsis(3) }}
                    >
                        {title}
                    </Typography>
                </Stack>
            </Box>
        </Tooltip>
    )
}