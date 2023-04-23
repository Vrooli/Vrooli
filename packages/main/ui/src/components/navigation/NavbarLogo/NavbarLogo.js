import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BUSINESS_NAME } from "@local/consts";
import { Box, Typography, useTheme } from "@mui/material";
export const NavbarLogo = ({ onClick, state, }) => {
    const { breakpoints, palette } = useTheme();
    if (state === "none")
        return null;
    return (_jsxs(Box, { onClick: onClick, sx: {
            padding: 0,
            display: "flex",
            alignItems: "center",
        }, children: [_jsx(Box, { sx: {
                    display: "flex",
                    padding: 0,
                    cursor: "pointer",
                    margin: "5px",
                    borderRadius: "500px",
                }, children: _jsx(Box, { component: "img", src: "assets/img/Logo-128x128.png", alt: `${BUSINESS_NAME} Logo`, sx: {
                        verticalAlign: "middle",
                        fill: "black",
                        marginLeft: "max(-5px, -5vw)",
                        width: "48px",
                        height: "48px",
                        [breakpoints.up("md")]: {
                            width: "6vh",
                            height: "6vh",
                        },
                    } }) }), state === "full" && _jsx(Typography, { variant: "h6", noWrap: true, sx: {
                    position: "relative",
                    cursor: "pointer",
                    lineHeight: "1.3",
                    fontSize: "3em",
                    fontFamily: "SakBunderan",
                    color: palette.primary.contrastText,
                }, children: BUSINESS_NAME })] }));
};
//# sourceMappingURL=NavbarLogo.js.map