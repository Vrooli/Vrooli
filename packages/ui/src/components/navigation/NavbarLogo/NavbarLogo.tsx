import { BUSINESS_NAME } from "@local/shared";
import { Box, IconButton, Typography, useTheme } from "@mui/material";
import { VrooliIcon } from "icons";
import { NavbarLogoProps } from "../types";


export const NavbarLogo = ({
    onClick,
    state,
}: NavbarLogoProps) => {
    const { palette } = useTheme();

    if (state === "none") return null;
    return (
        <Box
            onClick={onClick}
            sx={{
                padding: 0,
                display: "flex",
                alignItems: "center",
            }}
        >
            {/* Logo */}
            <IconButton sx={{
                display: "flex",
                padding: 0,
                margin: "5px",
                marginLeft: "max(-5px, -5vw)",
                width: "48px",
                height: "48px",
            }}>
                <VrooliIcon fill={palette.primary.contrastText} width="100%" height="100%" />
            </IconButton>
            {/* Business name */}
            {state === "full" && <Typography
                variant="h6"
                noWrap
                sx={{
                    position: "relative",
                    cursor: "pointer",
                    lineHeight: "1.3",
                    fontSize: "2.5em",
                    fontFamily: "SakBunderan",
                    color: palette.primary.contrastText,
                }}
            >{BUSINESS_NAME}</Typography>}
            {/* Alpha indicator */}
            <Typography
                variant="body2"
                noWrap
                sx={{
                    color: palette.error.main,
                    paddingLeft: state === "full" ? 1 : 0,
                }}
            >Alpha</Typography>

        </Box>
    );
};
