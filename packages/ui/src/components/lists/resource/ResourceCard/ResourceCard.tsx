import {
    Box,
    Stack,
    Tooltip,
    Typography,
} from '@mui/material';
import { firstString, getTranslation, getUserLanguages, openLink, PubSub, ResourceType, usePress } from 'utils';
import { useCallback, useMemo } from 'react';
import { useLocation } from '@shared/route';
import { ResourceCardProps } from '../../../cards/types';
import { containerShadow, multiLineEllipsis, noSelect } from 'styles';
import { getResourceIcon } from '..';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import { urlRegex, walletAddressRegex, adaHandleRegex } from '@shared/validation';
import { SnackSeverity, UsedForDisplay } from 'components/dialogs';

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
    index,
    onContextMenu,
    session,
}: ResourceCardProps) => {
    const [, setLocation] = useLocation();

    const { description, title } = useMemo(() => {
        const languages = getUserLanguages(session);
        const description = getTranslation(data, 'description', languages, true);
        const title = getTranslation(data, 'title', languages, true);
        return {
            description: (description && description.length > 0) ? description : data.link,
            title: (title && title.length > 0) ? title : UsedForDisplay[data.usedFor ?? ResourceUsedFor.Context],
        };
    }, [data, session]);

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link)
    }, [data]);

    const handleClick = useCallback((target: React.MouseEvent['target']) => {
        console.log('handle clickkkkkkkkkkk')
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
    const handleContextMenu = useCallback((target: React.MouseEvent['target']) => {
        if (onContextMenu && canEdit) onContextMenu(target, index);
    }, [onContextMenu, canEdit, index]);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        // onHover: handleContextMenu,
        onRightClick: handleContextMenu,
    });

    return (
        <Tooltip placement="top" title={`${description ? description + ' - ' : ''}${data.link}`}>
            <Box
                {...pressEvents}
                sx={{
                    ...noSelect,
                    ...containerShadow,
                    background: (t: any) => t.palette.primary.light,
                    color: (t: any) => t.palette.primary.contrastText,
                    borderRadius: '16px',
                    margin: 0,
                    padding: 1,
                    cursor: canEdit ? 'pointer' : 'default',
                    width: '120px',
                    minWidth: '120px',
                    minHeight: '120px',
                    height: '120px',
                    position: 'relative',
                    '&:hover': {
                        filter: canEdit ? `brightness(120%)` : 'none',
                        transition: 'filter 0.2s',
                    },
                } as any}
            >
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
                        {firstString(title, data.link)}
                    </Typography>
                </Stack>
            </Box>
        </Tooltip>
    )
}