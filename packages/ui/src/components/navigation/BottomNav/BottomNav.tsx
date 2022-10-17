import { useLocation } from '@shared/route';
import { BottomNavigation, useTheme } from '@mui/material';
import { actionsToBottomNav, ACTION_TAGS, getUserActions } from 'utils';
import { BottomNavProps } from '../types';

export const BottomNav = ({
    session,
    ...props
}: BottomNavProps) => {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();

    let actions = actionsToBottomNav({
        actions: getUserActions({ session, exclude: [ACTION_TAGS.Profile, ACTION_TAGS.LogIn] }),
        setLocation,
    });

    return (
        <BottomNavigation
            showLabels
            sx={{
                background: palette.primary.dark,
                position: 'fixed',
                zIndex: 5,
                bottom: 0,
                paddingBottom: 'env(safe-area-inset-bottom)',
                paddingLeft: 'env(safe-area-inset-left)',
                paddingRight: 'env(safe-area-inset-right)',
                height: 'calc(56px + env(safe-area-inset-bottom))',
                width: '100%',
                display: { xs: 'flex', md: 'none' },
            }}
            {...props}
        >
            {actions}
        </BottomNavigation>
    );
}