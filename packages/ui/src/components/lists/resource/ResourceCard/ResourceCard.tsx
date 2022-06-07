import {
    Box,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { getTranslation, openLink, Pubs, ResourceType } from 'utils';
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
import { urlRegex, walletAddressRegex, adaHandleRegex } from '@local/shared';
import { UsedForDisplay } from 'components/dialogs';

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

/**
 * Determines if a resource is a URL, wallet payment address, or an ADA handle
 * @param link String to check
 * @returns ResourceType if type found, or null if not
 */
const getResourceType = (link: string): ResourceType | null => {
    if (urlRegex.test(link)) return ResourceType.Url;
    if (walletAddressRegex.test(link)) return ResourceType.Wallet;
    if (adaHandleRegex.test(link)) return ResourceType.Handle;
    return null;
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
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { description, title } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        const description = getTranslation(data, 'description', languages, true);
        const title = getTranslation(data, 'title', languages, true);
        return {
            description: (description && description.length > 0) ? description : data.link,
            title: (title && title.length > 0) ? title : UsedForDisplay[data.usedFor ?? ResourceUsedFor.Context],
        };
    }, [data, session?.languages]);

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link)
    }, [data]);

    const handleClick = useCallback((event: any) => {
        // Find the resource type
        const resourceType = getResourceType(data.link);
        // If null, show error
        if (!resourceType) {
            PubSub.publish(Pubs.Snack, { message: 'Unable to open link', severity: 'error' });
            return;
        }
        // If URL, open in new tab
        if (resourceType === ResourceType.Url) openLink(setLocation, data.link);
        // If wallet address, open dialog to copy to clipboard
        else if (resourceType === ResourceType.Wallet) {
            PubSub.publish(Pubs.AlertDialog, {
                message: `Wallet address: ${data.link}`,
                buttons: [
                    {
                        text: 'Copy', onClick: () => {
                            navigator.clipboard.writeText(data.link);
                            PubSub.publish(Pubs.Snack, { message: 'Copied.', severity: 'success' });
                        }
                    },
                    { text: 'Close' }
                ]
            });
        }
        // If handle, open ADA Handle payment site
        else if (resourceType === ResourceType.Handle) openLink(setLocation, `https://handle.me/${data.link}`);
    }, [data.link, setLocation]);
    const handleRightClick = useCallback((event: any) => {
        if (onRightClick) onRightClick(event, index);
    }, [onRightClick, index]);

    const onEdit = useCallback((e) => {
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
                    minHeight: '120px',
                    height: '120px',
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
                        background: palette.error.main,
                        color: 'white',
                        transition: 'brightness 0.2s ease-in-out',
                        '&:hover': {
                            filter: `brightness(105%)`,
                            background: palette.error.main,
                        },
                    } as any}>
                        <DeleteIcon />
                    </IconButton>
                </Box>
                {/* Content */}
                <Stack
                    direction="column"
                    justifyContent="center"
                    alignItems="center"
                    sx={{
                        height: '100%',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                    }}
                >
                    <Icon sx={{ fill: 'white' }} />
                    <Typography
                        gutterBottom
                        variant="body2"
                        component="h3"
                        sx={{
                            ...multiLineEllipsis(3),
                            textAlign: 'center',
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