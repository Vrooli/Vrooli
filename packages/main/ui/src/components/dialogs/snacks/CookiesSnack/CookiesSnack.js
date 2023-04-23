import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CloseIcon, LargeCookieIcon } from "@local/icons";
import { Box, Button, Grid, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { useState } from "react";
import { noSelect } from "../../../../styles";
import { setCookiePreferences } from "../../../../utils/cookies";
import { CookieSettingsDialog } from "../../CookieSettingsDialog/CookieSettingsDialog";
export const CookiesSnack = ({ handleClose, }) => {
    const { palette } = useTheme();
    const [isCustomizeOpen, setIsCustomizeOpen] = useState(false);
    const handleAcceptAllCookies = () => {
        const preferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        setCookiePreferences(preferences);
        setIsCustomizeOpen(false);
        handleClose();
    };
    const handleCustomizeCookies = (preferences) => {
        if (preferences) {
            setCookiePreferences(preferences);
        }
        setIsCustomizeOpen(false);
        handleClose();
    };
    return (_jsxs(_Fragment, { children: [_jsx(CookieSettingsDialog, { handleClose: handleCustomizeCookies, isOpen: isCustomizeOpen }), _jsxs(Box, { sx: {
                    width: "min(100%, 500px)",
                    zIndex: 20000,
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    padding: 2,
                    borderRadius: 2,
                    boxShadow: 8,
                    pointerEvents: "auto",
                    ...noSelect,
                }, children: [_jsxs(Stack, { direction: "row", justifyContent: "space-between", alignItems: "center", children: [_jsx(LargeCookieIcon, { width: "80px", height: "80px", fill: palette.background.textPrimary }), _jsx(IconButton, { onClick: handleClose, children: _jsx(CloseIcon, { width: "32px", height: "32px", fill: palette.background.textPrimary }) })] }), _jsx(Typography, { variant: "body1", sx: { mt: 2 }, children: "This site uses cookies to give you the best experience. Please accept or reject our cookie policy so we can stop asking :)" }), _jsxs(Grid, { container: true, spacing: 2, sx: { mt: 2 }, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(Button, { fullWidth: true, color: "secondary", onClick: handleAcceptAllCookies, children: "Accept all cookies" }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(Button, { fullWidth: true, color: "secondary", variant: "outlined", onClick: () => setIsCustomizeOpen(true), children: "Customize cookies" }) })] })] })] }));
};
//# sourceMappingURL=CookiesSnack.js.map