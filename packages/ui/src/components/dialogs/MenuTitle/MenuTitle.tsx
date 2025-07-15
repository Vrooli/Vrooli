import Box from "@mui/material/Box";
import { styled, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { IconButton } from "../../buttons/IconButton.js";
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

const CloseButtonWrapper = styled(Box)({
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
            <CloseButtonWrapper>
                <IconButton
                    aria-label={t("Close")}
                    onClick={onClose}
                    onKeyDown={handleKeyDown}
                    variant="transparent"
                    size="md"
                >
                    <IconCommon
                        decorative
                        fill={palette.text.secondary}
                        name="Close"
                    />
                </IconButton>
            </CloseButtonWrapper>
        </TitleContainer>
    );
}
