import { BUSINESS_NAME } from ":local/consts";
import { Box, Typography, useTheme } from "@mui/material";
import { NavbarLogoProps } from "../types";


export const NavbarLogo = ({
    onClick,
    state,
}: NavbarLogoProps) => {
    const { breakpoints, palette } = useTheme();

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
            <Box sx={{
                display: "flex",
                padding: 0,
                cursor: "pointer",
                margin: "5px",
                borderRadius: "500px",
            }}>
                <Box
                    component="img"
                    src={"assets/img/Logo-128x128.png"}
                    alt={`${BUSINESS_NAME} Logo`}
                    sx={{
                        verticalAlign: "middle",
                        fill: "black",
                        marginLeft: "max(-5px, -5vw)",
                        width: "48px",
                        height: "48px",
                        [breakpoints.up("md")]: {
                            width: "6vh",
                            height: "6vh",
                        },
                    }}
                />
            </Box>
            {/* Business name */}
            {state === "full" && <Typography
                variant="h6"
                noWrap
                sx={{
                    position: "relative",
                    cursor: "pointer",
                    lineHeight: "1.3",
                    fontSize: "3em",
                    fontFamily: "SakBunderan",
                    color: palette.primary.contrastText,
                }}
            >{BUSINESS_NAME}</Typography>}
        </Box>
    );
};
