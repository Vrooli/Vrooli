import { Collapse, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
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
                aria-label="run-routine"
                component="button"
                onClick={toggleOpen}
                sx={{
                    background: palette.secondary.main,
                    padding: 0,
                    width: "54px",
                    height: "54px",
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
