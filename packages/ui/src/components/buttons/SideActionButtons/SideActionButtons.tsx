import { Stack } from "@mui/material"
import { SideActionButtonsProps } from "../types"

/**
 * Buttons displayed on bottom left or right of screen
 */
export const SideActionButtons = ({
    children,
    display,
    isLeftHanded,
    sx,
    zIndex,
}: SideActionButtonsProps) => {
    return (
        <Stack direction="row" spacing={2} sx={{
            position: 'fixed',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex,
            bottom: 0,
            // Buttons on right side of screen for right handed users, 
            // and left side of screen for left handed users
            left: isLeftHanded ? 0 : 'auto',
            right: isLeftHanded ? 'auto' : 0,
            // Make sure that spacing accounts for safe area insets, the height of the BottomNav, 
            // and action buttons that might be displayed above the BottomNav
            marginBottom: {
                // Only need to account for BottomNav when screen is small AND this is not for a dialog (which has no BottomNav)
                xs: display === 'page' ? 'calc(56px + 70px + 16px + env(safe-area-inset-bottom))' : 'calc(70px + 16px + env(safe-area-inset-bottom))',
                md: 'calc(70px + 16px + env(safe-area-inset-bottom))'
            },
            marginLeft: 'calc(16px + env(safe-area-inset-left))',
            marginRight: 'calc(16px + env(safe-area-inset-right))',
            height: 'calc(64px + env(safe-area-inset-bottom))',
            ...(sx ?? {}),
        }}>
            {children}
        </Stack>
    )
}