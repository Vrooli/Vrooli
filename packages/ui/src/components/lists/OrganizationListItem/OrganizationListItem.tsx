import { Box, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { OrganizationListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { StarFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading } from '..';
import { getTranslation, getUserLanguages, listItemColor, openObject, placeholderColor } from 'utils';
import { smallHorizontalScrollbar } from '../styles';
import { OrganizationIcon } from '@shared/icons';
import { ReportsButton, StarButton } from 'components/buttons';

export function OrganizationListItem({
    data,
    hideRole,
    index,
    loading,
    session,
    onClick,
    tooltip,
}: OrganizationListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { bio, canEdit, canStar, name, reportsCount } = useMemo(() => {
        const { canEdit, canStar } = data?.permissionsOrganization ?? {};
        const languages = getUserLanguages(session);
        const { bio, name } = getTranslation(data, languages, true);
        return { bio, canEdit, canStar, name, reportsCount: data?.reportsCount ?? 0 };
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the object's page
        else openObject(data, setLocation);
    }, [onClick, data, setLocation]);

    return (
        <Tooltip placement="top" title={tooltip ?? 'View Details'}>
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: listItemColor(index, palette),
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <Box
                        width="50px"
                        minWidth="50px"
                        height="50px"
                        borderRadius='100%'
                        bgcolor={profileColors[0]}
                        justifyContent='center'
                        alignItems='center'
                        sx={{
                            display: 'flex',
                        }}
                    >
                        <OrganizationIcon fill={profileColors[1]} width="35px" height="35px" />
                    </Box>
                    <Stack
                        direction="column"
                        spacing={1}
                        pl={2}
                        sx={{
                            width: '-webkit-fill-available',
                            display: 'grid',
                        }}
                    >
                        {/* Name/Title and role */}
                        {loading ? <TextLoading /> :
                            (
                                <Stack direction="row" spacing={1} sx={{
                                    ...smallHorizontalScrollbar(palette),
                                }}>
                                    <ListItemText
                                        primary={name}
                                        sx={{
                                            ...multiLineEllipsis(1),
                                            lineBreak: 'anywhere',
                                        }}
                                    />
                                    {!hideRole && canEdit && <ListItemText
                                        primary={`(Can Edit)`}
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            width: '100%',
                                            color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                            flex: 200,
                                        }}
                                    />}
                                </Stack>
                            )
                        }
                        {/* Bio/Description */}
                        {loading ? <TextLoading /> : <ListItemText
                            primary={bio}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />}
                        {/* Tags */}
                        {Array.isArray(data?.tags) && (data?.tags as any).length > 0 ?
                            <TagList
                                session={session}
                                parentId={data?.id ?? ''}
                                tags={data?.tags ?? []}
                                sx={{ ...smallHorizontalScrollbar(palette) }}
                            /> :
                            null}
                    </Stack>
                    {/* Star/Comment/Report */}
                    <Stack direction="column" spacing={1}>
                        <StarButton
                            disabled={!canStar}
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Organization}
                            isStar={data?.isStarred}
                            stars={data?.stars}
                        />
                        {reportsCount > 0 && <ReportsButton
                            reportsCount={data?.reportsCount ?? 0}
                            object={data}
                        />}
                    </Stack>
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}