import { Box, IconButton, Menu, Tooltip, useTheme } from "@mui/material";
import { MenuTitle } from "components/dialogs/MenuTitle/MenuTitle";
import { usePopover } from "hooks/usePopover";
import { HelpIcon } from "icons";
import React, { useCallback } from "react";
import { linkColors, noSelect } from "styles";
import { MarkdownDisplay } from "../../../../../../packages/ui/src/components/text/MarkdownDisplay/MarkdownDisplay";
import { HelpButtonProps } from "../types";

export const HelpButton = ({
    id = "help-details-menu",
    markdown,
    onClick,
    sxRoot,
    sx,
}: HelpButtonProps) => {
    const { palette } = useTheme();

    const [anchorEl, handleOpen, handleClose, isOpen] = usePopover();
    const openMenu = useCallback((event: React.MouseEvent<HTMLButtonElement>) => {
        if (onClick) onClick(event);
        if (!anchorEl) handleOpen(event);
    }, [anchorEl, handleOpen, onClick]);

    return (
        <Box
            sx={{
                display: "inline",
                ...sxRoot,
            }}
        >
            <Tooltip placement='top' title={!isOpen ? "Open Help Menu" : ""}>
                <IconButton
                    onClick={openMenu}
                    sx={{
                        display: "inline-flex",
                        bottom: "0",
                        verticalAlign: "top",
                    }}
                >
                    <HelpIcon fill={palette.secondary.main} {...sx} />
                    <Menu
                        id={id}
                        open={isOpen}
                        disableScrollLock={true}
                        anchorEl={anchorEl}
                        onClose={handleClose}
                        anchorOrigin={{
                            vertical: "bottom",
                            horizontal: "right",
                        }}
                        transformOrigin={{
                            vertical: "top",
                            horizontal: "left",
                        }}
                        sx={{
                            zIndex: "40000 !important",
                            "& .MuiPopover-paper": {
                                background: palette.background.default,
                                maxWidth: "min(90vw, 500px)",
                            },
                            "& .MuiMenu-list": {
                                padding: 0,
                            },
                        }}
                    >
                        <MenuTitle onClose={handleClose} />
                        <Box sx={{
                            padding: 2,
                            minHeight: "50px",
                            ...linkColors(palette),
                            ...noSelect,
                        }}>
                            <MarkdownDisplay content={markdown} />
                        </Box>
                    </Menu>
                </IconButton>
            </Tooltip>
        </Box>
    );
};
