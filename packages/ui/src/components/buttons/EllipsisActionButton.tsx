import Collapse from "@mui/material/Collapse";
import { IconButton } from "./IconButton.js";
import Stack from "@mui/material/Stack";
import { Tooltip } from "../Tooltip/Tooltip.js";
import { useTheme } from "@mui/material/styles";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { Icon } from "../../icons/Icons.js";
import { type EllipsisActionButtonProps } from "./types.js";

export function EllipsisActionButton({
    children,
}: EllipsisActionButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const isLeftHanded = useIsLeftHanded();

    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);

    const iconInfo = useMemo(() => {
        if (isOpen) return { name: "Close", type: "Common" } as const;
        return { name: "Ellipsis", type: "Common" } as const;
    }, [isOpen]);

    const button = useMemo(() => (
        <Tooltip title={t("MoreOptions")} placement="top">
            <IconButton
                variant="transparent"
                size={54}
                aria-label="run-routine"
                onClick={toggleOpen}
                style={{
                    background: palette.secondary.main,
                }}
            >
                <Icon
                    decorative
                    fill={palette.secondary.contrastText}
                    info={iconInfo}
                    size={36}
                />
            </IconButton>
        </Tooltip>
    ), [iconInfo, palette.secondary.contrastText, palette.secondary.main, toggleOpen, t]);

    return (
        <>
            {isLeftHanded && button}
            <Collapse orientation="horizontal" in={isOpen}>
                <Stack
                    spacing={{ xs: 1, sm: 1.5, md: 2 }}
                    direction="row"
                    alignItems="center"
                    justifyContent="center"
                    p={1}
                    sx={{
                        overflowX: "auto",
                        flexWrap: "wrap",
                        flexDirection: isLeftHanded ? "row-reverse" : "row",
                        justifyContent: isLeftHanded ? "left" : "right",
                    }}
                >
                    {children}
                </Stack>
            </Collapse>
            {!isLeftHanded && button}
        </>
    );
}
