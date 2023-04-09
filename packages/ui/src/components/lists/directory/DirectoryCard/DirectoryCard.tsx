import {
    Box,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { ApiIcon, DeleteIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { useCallback, useContext, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { multiLineEllipsis, noSelect } from 'styles';
import { getDisplay } from 'utils/display/listTools';
import { getUserLanguages } from 'utils/display/translationTools';
import usePress from 'utils/hooks/usePress';
import { getObjectUrl } from 'utils/navigation/openObject';
import { SessionContext } from 'utils/SessionContext';
import { DirectoryCardProps } from '../types';

/**
 * Unlike ResourceCard, these aren't draggable. This is because the objects 
 * are not stored in an order - they are stored by object type
 */
export const DirectoryCard = ({
    canUpdate,
    data,
    index,
    onContextMenu,
    onDelete,
}: DirectoryCardProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [showIcons, setShowIcons] = useState(false);

    const { title, subtitle } = useMemo(() => getDisplay(data as any, getUserLanguages(session)), [data, session]);

    const Icon = useMemo(() => {
        if (!data || !data.__typename) return HelpIcon;
        if (data.__typename === 'ApiVersion') return ApiIcon;
        if (data.__typename === 'NoteVersion') return NoteIcon;
        if (data.__typename === 'Organization') return OrganizationIcon;
        if (data.__typename === 'ProjectVersion') return ProjectIcon;
        if (data.__typename === 'RoutineVersion') return RoutineIcon;
        if (data.__typename === 'SmartContractVersion') return SmartContractIcon;
        if (data.__typename === 'StandardVersion') return StandardIcon;
        return HelpIcon;
    }, [data]);

    const href = useMemo(() => data ? getObjectUrl(data as any) : '#', [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Check if delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith('delete-')) {
            onDelete?.(index);
        }
        else {
            // Navigate to object
            setLocation(href);
        }
    }, [href, index, onDelete, setLocation]);
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
        <Tooltip placement="top" title={`${subtitle ? subtitle + ' - ' : ''}${href}`}>
            <Box
                {...pressEvents}
                component="a"
                href={href}
                onClick={(e) => e.preventDefault()}
                sx={{
                    ...noSelect,
                    boxShadow: 8,
                    background: palette.primary.light,
                    color: palette.secondary.contrastText,
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
                {/* delete icon, only visible on hover */}
                {showIcons && (
                    <>
                        <Tooltip title={t('Delete')}>
                            <ColorIconButton
                                id='delete-icon-button'
                                background={palette.error.main}
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
                    <Icon fill="white" />
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
                        {title}
                    </Typography>
                </Stack>
            </Box>
        </Tooltip>
    )
}