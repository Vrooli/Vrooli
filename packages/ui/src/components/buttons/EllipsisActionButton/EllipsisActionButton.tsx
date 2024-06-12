import { Collapse, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { CloseIcon, EllipsisIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { EllipsisActionButtonProps } from "../types";

export const EllipsisActionButton = ({
    children,
}: EllipsisActionButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const isLeftHanded = useIsLeftHanded();

    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);

    const Icon = useMemo(() => {
        if (isOpen) return CloseIcon;
        return EllipsisIcon;
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
                <Icon fill={palette.secondary.contrastText} width='36px' height='36px' />
            </IconButton>
        </Tooltip>
    ), [Icon, palette.secondary.contrastText, palette.secondary.main, toggleOpen, t]);

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
