import { useLocation } from '@shared/route';
import { BottomNavigation, useTheme } from '@mui/material';
import { actionsToBottomNav, getUserActions, useKeyboardOpen } from 'utils';
import { BottomNavProps } from '../types';

export const BottomNav = ({
    session,
    ...props
}: BottomNavProps) => {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();

    let actions = actionsToBottomNav({
        actions: getUserActions({ session }),
        setLocation,
    });

    // Hide the nav if the keyboard is open. This is because fixed bottom navs 
    // will appear above the keyboard on Android for some reason.
    const invisible = useKeyboardOpen();

    if (invisible) return null;
    return (
        <BottomNavigation
            showLabels
            sx={{
                background: palette.primary.dark,
                position: 'fixed',
                zIndex: 5,
                bottom: 0,
                paddingBottom: 'env(safe-area-inset-bottom)',
                paddingLeft: 'calc(4px + env(safe-area-inset-left))',
                paddingRight: 'calc(4px + env(safe-area-inset-right))',
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