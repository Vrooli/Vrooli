import { Box, IconButton, Stack, useTheme } from '@mui/material';
import { OpenThreadIcon, OrganizationIcon, UserIcon } from '@shared/icons';
import { useMemo } from 'react';
import { placeholderColor } from 'utils';
import { CommentConnectorProps } from '../types';

/**
 * Collapsible, vertical line for indicating a comment level. Top of line 
 * if the profile image of the comment.
 */
export const CommentConnector = ({
    isOpen,
    parentType,
    onToggle,
}: CommentConnectorProps) => {
    const { palette } = useTheme();

    // Random color for profile image (since we don't display custom image yet)
    const profileColors = useMemo(() => placeholderColor(), []);
    // Determine profile image type
    const ProfileIcon = useMemo(() => {
        switch (parentType) {
            case 'Organization':
                return OrganizationIcon;
            default:
                return UserIcon;
        }
    }, [parentType]);

    // Profile image
    const profileImage = useMemo(() => (
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
            <ProfileIcon fill={profileColors[1]} width='80%' height='80%' />
        </Box>
    ), [ProfileIcon, profileColors]);

    // If open, profile image on top of collapsible line
    if (isOpen) {
        return (
            <Stack direction="column">
                {/* Profile image */}
                {profileImage}
                {/* Collapsible, vertical line */}
                {
                    isOpen && <Box
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
                    />
                }
            </Stack>
        );
    }
    // If closed, OpenIcon to the left of profile image
    return (
        <Stack direction="row">
            <IconButton
                onClick={onToggle}
                sx={{
                    width: '48px',
                    height: '48px',
                }}
            >
                <OpenThreadIcon fill={profileColors[0]}/>
            </IconButton>
            {profileImage}
        </Stack>
    )
}