import {
    Box,
    IconButton,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import { getTranslation, openLink } from 'utils';
import { useCallback, useEffect, useMemo } from 'react';
import { useLocation } from 'wouter';
import { ResourceCardProps } from '../types';
import { cardRoot } from '../styles';
import { multiLineEllipsis, noSelect } from 'styles';
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
    onRightClick,
}: ResourceCardProps) => {
    const [, setLocation] = useLocation();

    const { description, title } = useMemo(() => {
        const languages = navigator.languages; //session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            title:  getTranslation(data, 'title', languages, true),
        };
    }, [data]);

    const handleClick = useCallback((event: any) => {
        if (data.link) openLink(setLocation, data.link);
    }, [data]);
    const handleRightClick = useCallback((event: any) => {
        if (onRightClick) onRightClick(event, data);
    }, [onRightClick, data]);

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
        <Tooltip placement="top" title={description ?? data.link}>
            <Box
                onClick={handleClick}
                onContextMenu={handleRightClick}
                sx={{
                    ...cardRoot,
                    ...noSelect,
                    padding: 1,
                    width: '120px',
                    minWidth: '120px',
                    position: 'relative'
                } as any}
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
                <Stack direction="column" justifyContent="center" alignItems="center" sx={{overflow: 'overlay'}}>
                    <Icon sx={{ fill: 'white' }} />
                    <Typography
                        gutterBottom
                        variant="body2"
                        component="h3"
                        sx={{ 
                            ...multiLineEllipsis(3),
                            lineBreak: Boolean(title) ? 'auto' : 'anywhere', // Line break anywhere only if showing link
                        }}
                    >
                        {title ?? data.link}
                    </Typography>
                </Stack>
            </Box>
        </Tooltip>
    )
}