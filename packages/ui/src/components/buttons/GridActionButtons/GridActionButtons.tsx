/**
 * Prompts user to select which link the new node should be added on
 */
import { Grid, useTheme } from '@mui/material';
import { GridActionButtonsProps } from '../types';

export const GridActionButtons = ({
    children,
    display,
}: GridActionButtonsProps) => {
    const { palette } = useTheme();

    return (
        <Grid container spacing={2} sx={{
            padding: 2,
            paddingTop: 0,
            marginLeft: 'auto',
            marginRight: 'auto',
            maxWidth: 'min(700px, 100%)',
            left: display === 'page' ? undefined : 0,
            zIndex: 1,
            // Position is sticky when used for a page or for large screens, and static when used for a dialog
            position: { xs: display === 'page' ? 'sticky' : 'fixed', sm: 'sticky' },
            // Displayed directly above BottomNav (pages only), which is only visible on mobile
            bottom: { xs: display === 'page' ? 'calc(56px + env(safe-area-inset-bottom))' : 0, md: 0 },
            paddingBottom: display === 'page' ? undefined : 'calc(12px + env(safe-area-inset-bottom))',
            // Background has transparent blur gradient when used for a page, 
            // and a solid color when used for a dialog
            background: display === 'page' ? 'transparent' : palette.primary.dark,
            backdropFilter: display === 'page' ? 'blur(5px)' : undefined,
        }}
        >
            {children}
        </Grid>
    )
}