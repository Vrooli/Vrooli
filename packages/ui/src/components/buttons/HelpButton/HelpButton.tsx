import { Box, Button, IconButton, Menu, Tooltip, useTheme } from "@mui/material";
import { usePopover } from "hooks/usePopover";
import { HelpIcon } from "icons";
import React, { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { linkColors, noSelect } from "styles";
import { MenuTitle } from "../../dialogs/MenuTitle/MenuTitle";
import { RichInputBase } from "../../inputs/RichInput/RichInput";
import { MarkdownDisplay } from "../../text/MarkdownDisplay/MarkdownDisplay";
import { HelpButtonProps } from "../types";

export function HelpButton({
    id = "help-details-menu",
    markdown,
    onMarkdownChange,
    onClick,
    sxRoot,
    sx,
}: HelpButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [anchorEl, handleOpen, handleClose, isOpen] = usePopover();
    const openMenu = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (onClick) onClick(event);
        if (!anchorEl) handleOpen(event);
    }, [anchorEl, handleOpen, onClick]);

    const [editedMarkdown, setEditedMarkdown] = useState(markdown);
    function handleSave() {
        onMarkdownChange?.(editedMarkdown);
        handleClose();
    }
    function handleCancel() {
        setEditedMarkdown(markdown);
        handleClose();
    }

    return (
        <Box
            sx={{
                display: "inline",
                ...sxRoot,
            }}
        >
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
                {typeof onMarkdownChange === "function" ? (<div>
                    <RichInputBase
                        maxRows={10}
                        minRows={4}
                        name="helpText"
                        onChange={setEditedMarkdown}
                        placeholder="Enter help text..."
                        value={editedMarkdown}
                        sxs={{ topBar: { borderRadius: 0 } }}
                    />
                    <Button onClick={handleSave} color="primary">{t("Save")}</Button>
                    <Button onClick={handleCancel} color="secondary">{t("Cancel")}</Button>
                </div>) : (<Box sx={{
                    paddingLeft: 2,
                    paddingRight: 2,
                    // Markdown elements themselves typically have top-bottom padding, so we don't need to add it here
                    paddingTop: 0,
                    paddingBottom: 0,
                    minHeight: "50px",
                    ...linkColors(palette),
                    ...noSelect,
                }}>
                    <MarkdownDisplay content={markdown} />
                </Box>)}
            </Menu>
            <Tooltip placement='top' title={!isOpen ? "Open Help Menu" : ""}>
                <Box onClick={openMenu}>
                    {typeof onMarkdownChange === "function" && markdown.length === 0 ?
                        (<Button variant="text" sx={{ textDecoration: "underline", textTransform: "none" }}>
                            Help text
                        </Button>) :
                        (<IconButton
                            sx={{
                                display: "inline-flex",
                                bottom: "0",
                                verticalAlign: "top",
                            }}
                        >
                            <HelpIcon fill={palette.secondary.main} {...sx} />
                        </IconButton>)
                    }
                </Box>
            </Tooltip>
        </Box>
    );
}
