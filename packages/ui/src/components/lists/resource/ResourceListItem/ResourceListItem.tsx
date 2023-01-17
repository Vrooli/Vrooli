// Used to display popular/search results of a particular object type
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { ResourceListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { ResourceUsedFor } from '@shared/consts';
import { adaHandleRegex, urlRegex, walletAddressRegex } from '@shared/validation';
import { useLocation } from '@shared/route';
import { firstString, getDisplay, getResourceUrl, getUserLanguages, openLink, PubSub, ResourceType, usePress } from 'utils';
import { getResourceIcon } from '..';
import { SnackSeverity, TextLoading } from 'components';
import { DeleteIcon, EditIcon, OpenInNewIcon } from '@shared/icons';

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

export function ResourceListItem({
    canEdit,
    data,
    handleContextMenu,
    handleDelete,
    handleEdit,
    index,
    loading,
    session,
}: ResourceListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);

    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Ignore if clicked edit or delete button
        if (target.id && ['delete-icon-button', 'edit-icon-button'].includes(target.id)) return;
        // If no resource type or link, show error
        const resourceType = getResourceType(data.link);
        if (!resourceType || !href) {
            PubSub.get().publishSnack({ messageKey: 'CannotOpenLink', severity: SnackSeverity.Error });
            return;
        }
        // Open link
        else openLink(setLocation, href);
    }, [data.link, href, setLocation]);

    const onEdit = useCallback((e: any) => {
        console.log('onEdit', e);
        handleEdit(index);
    }, [handleEdit, index]);

    const onDelete = useCallback(() => {
        handleDelete(index);
    }, [handleDelete, index]);

    const pressEvents = usePress({
        onLongPress: (target) => { handleContextMenu(target, index) },
        onClick: handleClick,
        onRightClick: (target) => { handleContextMenu(target, index) },
    });

    return (
        <Tooltip placement="top" title="Open in new tab">
            <ListItem
                disablePadding
                {...pressEvents}
                onClick={(e) => { e.preventDefault(); }}
                component="a"
                href={href}
                sx={{
                    display: 'flex',
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    borderBottom: `1px solid ${palette.divider}`,
                    padding: 1,
                    cursor: 'pointer',
                }}
            >
                <IconButton sx={{
                    width: "48px",
                    height: "48px",
                }}>
                    <Icon fill={palette.background.textPrimary} width="80%" height="80%" />
                </IconButton>
                <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                    {/* Name/Title */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={firstString(title, data.link)}
                        sx={{ ...multiLineEllipsis(1) }}
                    />}
                    {/* Bio/Description */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={subtitle}
                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                    />}
                </Stack>
                {
                    canEdit && <IconButton id='delete-icon-button' onClick={onDelete}>
                        <DeleteIcon fill={palette.background.textPrimary} />
                    </IconButton>
                }
                {
                    canEdit && <IconButton id='edit-icon-button' onClick={onEdit}>
                        <EditIcon fill={palette.background.textPrimary} />
                    </IconButton>
                }
                <IconButton>
                    <OpenInNewIcon fill={palette.background.textPrimary} />
                </IconButton>
            </ListItem>
        </Tooltip>
    )
}