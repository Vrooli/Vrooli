import { Box, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { UserListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { StarFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { getTranslation, getUserLanguages, listItemColor, openObject, placeholderColor } from 'utils';
import { TextLoading } from '../TextLoading/TextLoading';
import { smallHorizontalScrollbar } from '../styles';
import { UserIcon } from '@shared/icons';
import { ReportsButton, StarButton } from 'components/buttons';
import { getCurrentUser } from 'utils/authentication';

export const UserListItem = ({
    data,
    index,
    loading,
    onClick,
    session,
    tooltip = 'View details',
}: UserListItemProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isOwn = useMemo(() => data?.id === getCurrentUser(session).id, [data, session]);

    const profileColors = useMemo(() => placeholderColor(), []);

    const { bio, name, reportsCount } = useMemo(() => {
        const languages = getUserLanguages(session);
        return {
            bio: getTranslation(data, 'bio', languages, true),
            name: data?.name ?? (data?.handle ? `$${data.handle}` : ''),
            reportsCount: data?.reportsCount ?? 0,
        }
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the user's profile
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
                        alignItems='center'
                        bgcolor={profileColors[0]}
                        borderRadius='100%'
                        height="50px"
                        justifyContent='center'
                        minWidth="50px"
                        width="50px"
                        sx={{
                            display: 'flex',
                        }}
                    >
                        <UserIcon fill={profileColors[1]} width="35px" height="35px" />
                    </Box>
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
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
                                    {isOwn && <ListItemText
                                        primary={`(You)`}
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
                        {loading ? <TextLoading /> : <ListItemText
                            primary={bio}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />}
                    </Stack>
                    {/* Star/Comment/Report */}
                    <Stack direction="column" spacing={1}>
                        <StarButton
                            disabled={isOwn}
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.User}
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