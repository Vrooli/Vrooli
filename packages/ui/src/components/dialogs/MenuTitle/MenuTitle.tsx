import { Box, IconButton, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { CloseIcon } from "icons";
import { noSelect } from "styles";
import { MenuTitleProps } from "../types";

export const MenuTitle = ({
    ariaLabel,
    helpText,
    onClose,
    title,
    zIndex,
}: MenuTitleProps) => {
    const { palette } = useTheme();

    return (
        <Box
            id={ariaLabel}
            sx={{
                ...noSelect,
                display: "flex",
                alignItems: "center",
                paddingTop: 1,
                paddingBottom: 1,
                paddingLeft: 2,
                paddingRight: 2,
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                textAlign: "center",
                fontSize: { xs: "1.5rem", sm: "2rem" },
            }}
        >
            <Box sx={{ marginLeft: "auto" }} >{title}</Box>
            {helpText && <HelpButton
                markdown={helpText}
                sx={{
                    fill: palette.secondary.light,
                }}
                sxRoot={{
                    display: "flex",
                    marginTop: "auto",
                    marginBottom: "auto",
                }}
                zIndex={zIndex}
            />}
            <IconButton
                aria-label="close"
                edge="end"
                onClick={onClose}
                onKeyDown={(e) => {
                    // Cancel tabbing to close button. 
                    // Useful for menus with tabbable items, like ListMenu
                    if (e.key === "Tab") {
                        e.stopPropagation();
                    }
                }}
                sx={{ marginLeft: "auto" }}
            >
                <CloseIcon fill={palette.primary.contrastText} />
            </IconButton>
        </Box>
    );
};
