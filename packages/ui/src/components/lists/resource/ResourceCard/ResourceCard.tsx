import {
    Box,
    IconButton,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import { getTranslation, openLink } from 'utils';
import { useCallback, useMemo } from 'react';
import { useLocation } from 'wouter';
import { ResourceCardProps } from '../../../cards/types';
import { cardRoot } from '../../../cards/styles';
import { multiLineEllipsis, noSelect } from 'styles';
import {
    Delete as DeleteIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { getResourceIcon } from '..';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';

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
    canEdit,
    data,
    handleEdit,
    handleDelete,
    index,
    onRightClick,
    session,
}: ResourceCardProps) => {
    const [, setLocation] = useLocation();

    const { description, title } = useMemo(() => {
        console.log('getting resource translations', data)
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            title: getTranslation(data, 'title', languages, true),
        };
    }, [data]);
    console.log('resource ', title)

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link)
    }, [data]);

    const handleClick = useCallback((event: any) => {
        if (data.link) openLink(setLocation, data.link);
    }, [data]);
    const handleRightClick = useCallback((event: any) => {
        if (onRightClick) onRightClick(event, index);
    }, [onRightClick, index]);

    const onEdit = useCallback((e) => {
        console.log('on edittttttt')
        e.stopPropagation();
        handleEdit(index);
    }, [handleEdit, index]);

    const onDelete = useCallback((e) => {
        e.stopPropagation();
        handleDelete(index);
    }, [handleDelete, index]);

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
                        opacity: canEdit ? '1' : '0',
                    },
                }}>
                    <IconButton size="small" onClick={onEdit} aria-label="close" sx={{
                        ...buttonProps,
                        left: '-15px',
                        background: '#c5ab17',
                        color: 'white',
                        transition: 'brightness 0.2s ease-in-out',
                        '&:hover': {
                            filter: `brightness(105%)`,
                            background: '#c5ab17',
                        },
                    } as any}>
                        <EditIcon />
                    </IconButton>
                    <IconButton size="small" onClick={onDelete} aria-label="close" sx={{
                        ...buttonProps,
                        right: '-15px',
                        background: (t) => t.palette.error.main,
                        color: 'white',
                        transition: 'brightness 0.2s ease-in-out',
                        '&:hover': {
                            filter: `brightness(105%)`,
                            background: (t) => t.palette.error.main,
                        },
                    } as any}>
                        <DeleteIcon />
                    </IconButton>
                </Box>
                {/* Content */}
                <Stack direction="column" justifyContent="center" alignItems="center" sx={{ overflow: 'overlay' }}>
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