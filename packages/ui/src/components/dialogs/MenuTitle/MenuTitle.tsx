import { Box, IconButton, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { noSelect } from "../../../styles.js";
import { Title } from "../../text/Title.js";
import { MenuTitleProps } from "../types.js";

export function MenuTitle({
    ariaLabel,
    onClose,
    ...titleData
}: MenuTitleProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    function handleKeyDown(event: React.KeyboardEvent<HTMLButtonElement>) {
        // Cancel tabbing to close button. 
        // Useful for menus with tabbable items, like ListMenu
        if (event.key === "Tab") {
            event.stopPropagation();
        }
    }

    return (
        <Box
            id={ariaLabel}
            sx={{
                ...noSelect,
                display: "flex",
                alignItems: "center",
                paddingTop: 0,
                paddingBottom: 0,
                paddingLeft: 2,
                paddingRight: 2,
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                textAlign: "center",
                fontSize: { xs: "1.5rem", sm: "2rem" },
                ...titleData.sxs?.root,
            }}
        >
            {/* <Box sx={{ marginLeft: "auto" }} >{title}</Box>
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
            />} */}
            <Title
                variant="subheader"
                {...titleData}
                sxs={{
                    stack: { marginLeft: "auto", padding: 1 },
                }}
            />
            <IconButton
                aria-label={t("Close")}
                edge="end"
                onClick={onClose}
                onKeyDown={handleKeyDown}
                sx={{ marginLeft: "auto" }}
            >
                <IconCommon
                    decorative
                    fill={palette.primary.contrastText}
                    name="Close"
                />
            </IconButton>
        </Box>
    );
}
