import { BUSINESS_NAME, VrooliIcon } from "@local/shared";
import { Box, IconButton, Typography, useTheme } from "@mui/material";
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
        </Box>
    );
};
