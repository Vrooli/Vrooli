import { Box, IconButton, Menu, Tooltip, useTheme } from "@mui/material";
import { MenuTitle } from "components/dialogs/MenuTitle/MenuTitle";
import { HelpIcon } from "icons";
import { useCallback, useState } from "react";
import { linkColors, noSelect } from "styles";
import { MarkdownDisplay } from "../../../../../../packages/ui/src/components/text/MarkdownDisplay/MarkdownDisplay";
import { HelpButtonProps } from "../types";

export const HelpButton = ({
    id = "help-details-menu",
    markdown,
    onClick,
    sxRoot,
    sx,
    zIndex,
}: HelpButtonProps) => {
    const { palette } = useTheme();
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = Boolean(anchorEl);

    const openMenu = useCallback((event: any) => {
        if (onClick) onClick(event);
        if (!anchorEl) setAnchorEl(event.currentTarget);
    }, [anchorEl, onClick]);
    const closeMenu = () => {
        setAnchorEl(null);
    };

    return (
        <Box
            sx={{
                display: "inline",
                ...sxRoot,
            }}
        >
            <Tooltip placement='top' title={!open ? "Open Help Menu" : ""}>
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
                        open={open}
                        disableScrollLock={true}
                        anchorEl={anchorEl}
                        onClose={closeMenu}
                        anchorOrigin={{
                            vertical: "bottom",
                            horizontal: "right",
                        }}
                        transformOrigin={{
                            vertical: "top",
                            horizontal: "left",
                        }}
                        sx={{
                            "& .MuiPopover-paper": {
                                background: palette.background.default,
                                maxWidth: "min(90vw, 500px)",
                            },
                            "& .MuiMenu-list": {
                                padding: 0,
                            },
                        }}
                    >
                        <MenuTitle onClose={closeMenu} zIndex={zIndex} />
                        <Box sx={{ padding: 2, ...linkColors(palette), ...noSelect }}>
                            <MarkdownDisplay content={markdown} zIndex={zIndex} />
                        </Box>
                    </Menu>
                </IconButton>
            </Tooltip>
        </Box>
    );
};
