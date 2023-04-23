import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Button, Typography, useTheme } from "@mui/material";
export const TIDCard = ({ buttonText, description, key, Icon, onClick, title, }) => {
    const { palette } = useTheme();
    return (_jsxs(Box, { onClick: onClick, sx: {
            width: "100%",
            boxShadow: 8,
            padding: 1,
            borderRadius: 2,
            cursor: "pointer",
            background: palette.background.paper,
            "&:hover": {
                filter: "brightness(1.1)",
                boxShadow: 12,
            },
            display: "flex",
        }, children: [Icon && _jsx(Box, { sx: {
                    width: "75px",
                    height: "100%",
                    padding: 2,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                }, children: _jsx(Icon, { width: "50px", height: "50px", fill: palette.background.textPrimary }) }), _jsxs(Box, { sx: {
                    flexGrow: 1,
                    height: "100%",
                    padding: "1rem",
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "space-between",
                }, children: [_jsxs(Box, { children: [_jsx(Typography, { variant: 'h6', component: 'div', children: title }), _jsx(Typography, { variant: 'body2', color: palette.background.textSecondary, children: description })] }), _jsx(Button, { size: 'small', sx: {
                            marginTop: 2,
                            marginLeft: "auto",
                            alignSelf: "flex-end",
                        }, children: buttonText })] })] }, key));
};
//# sourceMappingURL=TIDCard.js.map