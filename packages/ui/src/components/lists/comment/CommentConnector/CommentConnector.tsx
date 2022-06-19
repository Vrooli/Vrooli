import {
    Apartment as OrganizationIcon,
    Person as UserIcon,
    UnfoldMore as OpenIcon,
} from '@mui/icons-material';
import { Box, IconButton, Stack, useTheme } from '@mui/material';
import { useMemo } from 'react';
import { ObjectType, placeholderColor } from 'utils';
import { CommentConnectorProps } from '../types';

/**
 * Collapsible, vertical line for indicating a comment level. Top of line 
 * if the profile image of the comment.
 */
export const CommentConnector = ({
    isOpen,
    objectType,
    onToggle,
}: CommentConnectorProps) => {
    const { palette } = useTheme();

    // Random color for profile image (since we don't display custom image yet)
    const profileColors = useMemo(() => placeholderColor(), []);
    // Determine profile image type
    const ProfileIcon = useMemo(() => {
        switch (objectType) {
            case ObjectType.Organization:
                return OrganizationIcon;
            default:
                return UserIcon;
        }
    }, [objectType]);

    return (
        // Profile image on top of collapsible line
        <Stack direction="column">
            {/* Profile image */}
            <Box
                width="48px"
                minWidth="48px"
                height="48px"
                minHeight="48px"
                borderRadius='100%'
                bgcolor={profileColors[0]}
                justifyContent='center'
                alignItems='center'
                sx={{
                    display: 'flex',
                }}
            >
                <ProfileIcon sx={{
                    fill: profileColors[1],
                    width: '80%',
                    height: '80%',
                }} />
            </Box>
            {/* Collapsible, vertical line if open */}
            {isOpen && <Box
                width="5px"
                height="100%"
                borderRadius='100px'
                bgcolor={profileColors[0]}
                sx={{
                    marginLeft: 'auto',
                    marginRight: 'auto',
                    marginTop: 1,
                    marginBottom: 1,
                    cursor: 'pointer',
                    '&:hover': {
                        brightness: palette.mode === 'light' ? 1.05 : 0.95,
                    },
                }}
                onClick={onToggle}
            />}
            {/* OpenIcon if closed */}
            {!isOpen && <IconButton
                onClick={onToggle}
                sx={{
                    width: '48px',
                    height: '48px',
                }}
            >
                <OpenIcon />
            </IconButton>}
        </Stack>
    )
}