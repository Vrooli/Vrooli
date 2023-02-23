import {
    Box,
    Stack,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { firstString, getDisplay, getResourceIcon, getResourceType, getResourceUrl, getUserLanguages, PubSub, usePress } from 'utils';
import { useCallback, useMemo, useState } from 'react';
import { openLink, useLocation } from '@shared/route';
import { ResourceCardProps } from '../types';
import { multiLineEllipsis, noSelect } from 'styles';
import { DeleteIcon, EditIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';
import { ResourceUsedFor } from '@shared/consts';
import { useTranslation } from 'react-i18next';

export const ResourceCard = ({
    canUpdate,
    data,
    index,
    onContextMenu,
    onEdit,
    onDelete,
    session,
}: ResourceCardProps) => {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [showIcons, setShowIcons] = useState(false);

    const { title, subtitle } = useMemo(() => {
        const { title, subtitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: Boolean(title) ? title : t(`common:${data.usedFor ?? ResourceUsedFor.Context}`, { lng: getUserLanguages(session)[0] }),
            subtitle,
        };
    }, [data, session, t]);

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link)
    }, [data]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Check if edit or delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith('edit-')) {
            onEdit?.(index);
        }
        else if (targetId && targetId.startsWith('delete-')) {
            onDelete?.(index);
        }
        else {
            // If no resource type or link, show error
            const resourceType = getResourceType(data.link);
            if (!resourceType || !href) {
                PubSub.get().publishSnack({ messageKey: 'CannotOpenLink', severity: 'Error' });
                return;
            }
            // Open link
            else openLink(setLocation, href);
        }
    }, [data.link, href, index, onDelete, onEdit, setLocation]);
    const handleContextMenu = useCallback((target: EventTarget) => {
        onContextMenu(target, index);
    }, [onContextMenu, index]);

    const handleHover = useCallback(() => {
        if (canUpdate) {
            setShowIcons(true);
        }
    }, [canUpdate]);

    const handleHoverEnd = useCallback(() => { setShowIcons(false) }, []);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onHover: handleHover,
        onHoverEnd: handleHoverEnd,
        onRightClick: handleContextMenu,
        hoverDelay: 100,
    });

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + ' - ' : ''}${data.link}`}>
            <Box
                {...pressEvents}
                component="a"
                href={href}
                onClick={(e) => e.preventDefault()}
                sx={{
                    ...noSelect,
                    boxShadow: 8,
                    background: (t: any) => t.palette.primary.light,
                    color: (t: any) => t.palette.primary.contrastText,
                    borderRadius: '16px',
                    margin: 0,
                    padding: 1,
                    cursor: 'pointer',
                    width: '120px',
                    minWidth: '120px',
                    minHeight: '120px',
                    height: '120px',
                    position: 'relative',
                    '&:hover': {
                        filter: `brightness(120%)`,
                        transition: 'filter 0.2s',
                    },
                } as any}
            >
                {/* Edit and delete icons, only visible on hover */}
                {showIcons && (
                    <>
                        <Tooltip title="Edit">
                            <ColorIconButton
                                id='edit-icon-button'
                                background={palette.secondary.main}
                                sx={{ position: 'absolute', top: 4, left: 4 }}
                            >
                                <EditIcon id='edit-icon' fill={palette.secondary.contrastText} />
                            </ColorIconButton>
                        </Tooltip>
                        <Tooltip title="Delete">
                            <ColorIconButton
                                id='delete-icon-button'
                                background={palette.secondary.main}
                                sx={{ position: 'absolute', top: 4, right: 4 }}
                            >
                                <DeleteIcon id='delete-icon' fill={palette.secondary.contrastText} />
                            </ColorIconButton>
                        </Tooltip>
                    </>
                )}
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