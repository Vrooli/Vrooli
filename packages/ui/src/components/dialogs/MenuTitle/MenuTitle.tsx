import { Box, IconButton, styled, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { noSelect } from "../../../styles.js";
import { Title } from "../../text/Title.js";
import { type MenuTitleProps } from "../types.js";

const TitleContainer = styled(Box)(({ theme }) => ({
    ...noSelect,
    display: "flex",
    alignItems: "center",
    paddingTop: 0,
    paddingBottom: 0,
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    color: theme.palette.text.primary,
    textAlign: "center",
}));

const CloseButton = styled(IconButton)({
    marginLeft: "auto",
});

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
        <TitleContainer
            id={ariaLabel}
            sx={titleData.sxs?.root}
        >
            <Title
                variant="subheader"
                {...titleData}
            />
            <CloseButton
                aria-label={t("Close")}
                edge="end"
                onClick={onClose}
                onKeyDown={handleKeyDown}
            >
                <IconCommon
                    decorative
                    fill={palette.text.secondary}
                    name="Close"
                />
            </CloseButton>
        </TitleContainer>
    );
}
