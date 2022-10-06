// Used to display popular/search results of a particular object type
import { IconButton, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { ResourceListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { ResourceSortBy, ResourceUsedFor } from '@shared/consts';
import { adaHandleRegex, urlRegex, walletAddressRegex } from '@shared/validation';
import { useLocation } from '@shared/route';
import { firstString, getTranslation, LabelledSortOption, labelledSortOptions, listItemColor, openLink, PubSub, ResourceType } from 'utils';
import { Resource } from 'types';
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
    handleDelete,
    handleEdit,
    index,
    loading,
    session,
}: ResourceListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { description, title } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            title: getTranslation(data, 'title', languages, true),
        };
    }, [data, session]);

    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);

    const handleClick = useCallback((event: any) => {
        event.stopPropagation();
        // Find the resource type
        const resourceType = getResourceType(data.link);
        // If null, show error
        if (!resourceType) {
            PubSub.get().publishSnack({ message: 'Unable to open link', severity: SnackSeverity.Error });
            return;
        }
        // If URL, open in new tab
        if (resourceType === ResourceType.Url) openLink(setLocation, data.link);
        // If wallet address, open dialog to copy to clipboard
        else if (resourceType === ResourceType.Wallet) {
            PubSub.get().publishAlertDialog({
                message: `Wallet address: ${data.link}`,
                buttons: [
                    {
                        text: 'Copy', onClick: () => {
                            navigator.clipboard.writeText(data.link);
                            PubSub.get().publishSnack({ message: 'Copied.', severity: SnackSeverity.Success });
                        }
                    },
                    { text: 'Close' }
                ]
            });
        }
        // If handle, open ADA Handle payment site
        else if (resourceType === ResourceType.Handle) openLink(setLocation, `https://handle.me/${data.link}`);
    }, [data.link, setLocation]);

    const onEdit = useCallback((e) => {
        e.stopPropagation();
        handleEdit(index);
    }, [handleEdit, index]);

    const onDelete = useCallback((e) => {
        e.stopPropagation();
        handleDelete(index);
    }, [handleDelete, index]);

    return (
        <Tooltip placement="top" title="Open in new tab">
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: listItemColor(index, palette),
                    color: palette.background.textPrimary,
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
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
                            primary={description}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />}
                    </Stack>
                    {
                        canEdit && <IconButton onClick={onDelete}>
                            <DeleteIcon fill={palette.background.textPrimary} />
                        </IconButton>
                    }
                    {
                        canEdit && <IconButton onClick={onEdit}>
                            <EditIcon fill={palette.background.textPrimary} />
                        </IconButton>
                    }
                    <IconButton>
                        <OpenInNewIcon fill={palette.background.textPrimary} />
                    </IconButton>
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const ResourceSortOptions: LabelledSortOption<ResourceSortBy>[] = labelledSortOptions(ResourceSortBy);
export const resourceDefaultSortOption = ResourceSortOptions[1];
export const resourceOptionLabel = (o: Resource, languages: readonly string[]) => getTranslation(o, 'title', languages, true) ?? '';