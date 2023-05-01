import { CloseIcon, EllipsisIcon } from "@local/shared";
import { Collapse, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { ColorIconButton } from "../ColorIconButton/ColorIconButton";
import { EllipsisActionButtonProps } from "../types";

export function EllipsisActionButton({
    children,
}: EllipsisActionButtonProps) {
    const { palette } = useTheme();

    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);

    const Icon = useMemo(() => {
        if (isOpen) return CloseIcon;
        return EllipsisIcon;
    }, [isOpen]);

    return (
        <>
            <Collapse orientation="horizontal" in={isOpen}>
                <Stack
                    spacing={{ xs: 1, sm: 1.5, md: 2 }}
                    direction="row"
                    alignItems="center"
                    justifyContent="center"
                    p={1}
                    sx={{
                        overflowX: "auto",
                    }}
                >
                    {children}
                </Stack>
            </Collapse>
            <Tooltip title="More options" placement="top">
                <ColorIconButton
                    aria-label="run-routine"
                    background={palette.secondary.main}
                    onClick={toggleOpen}
                    sx={{
                        padding: 0,
                        width: "54px",
                        height: "54px",
                    }}
                >
                    <Icon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
            </Tooltip>
        </>
    );
}
