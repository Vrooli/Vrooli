import { Box, Button, IconButton, Menu, Tooltip, styled, useTheme } from "@mui/material";
import React, { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { usePopover } from "../../../hooks/usePopover.js";
import { HelpIcon } from "../../../icons/common.js";
import { linkColors, noSelect } from "../../../styles.js";
import { MenuTitle } from "../../dialogs/MenuTitle/MenuTitle.js";
import { RichInputBase } from "../../inputs/RichInput/RichInput.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { HelpButtonProps } from "../types.js";

const HelpMenu = styled(Menu)(() => ({
    zIndex: "40000 !important",
    "& .MuiPopover-paper": {
        maxWidth: "min(90vw, 500px)",
    },
    "& .MuiMenu-list": {
        padding: 0,
    },
}));
const DisplayBox = styled(Box)(({ theme }) => ({
    padding: theme.spacing(2),
    minHeight: "50px",
    ...linkColors(theme.palette),
    ...noSelect,
}));

const anchorOrigin = {
    vertical: "bottom",
    horizontal: "right",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "left",
} as const;
const helpIconButtonStyle = {
    display: "inline-flex",
    bottom: "0",
    verticalAlign: "top",
} as const;
const helpTextButtonStyle = {
    textDecoration: "underline",
    textTransform: "none",
} as const;
const richInputBaseStyle = {
    topBar: { borderRadius: 0 },
} as const;

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

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            display: "inline",
            ...sxRoot,
        } as const;
    }, [sxRoot]);

    return (
        <Box sx={outerBoxStyle}>
            <HelpMenu
                id={id}
                open={isOpen}
                disableScrollLock={true}
                anchorEl={anchorEl}
                onClose={handleClose}
                anchorOrigin={anchorOrigin}
                transformOrigin={transformOrigin}
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
                        sxs={richInputBaseStyle}
                    />
                    <Button onClick={handleSave} color="primary">{t("Save")}</Button>
                    <Button onClick={handleCancel} color="secondary">{t("Cancel")}</Button>
                </div>) : (<DisplayBox>
                    <MarkdownDisplay content={markdown} />
                </DisplayBox>)}
            </HelpMenu>
            <Tooltip placement='top' title={!isOpen ? "Open Help Menu" : ""}>
                <Box onClick={openMenu}>
                    {typeof onMarkdownChange === "function" && markdown.length === 0 ?
                        (<Button variant="text" sx={helpTextButtonStyle}>
                            Help text
                        </Button>) :
                        (<IconButton sx={helpIconButtonStyle}>
                            <HelpIcon fill={palette.secondary.main} {...sx} />
                        </IconButton>)
                    }
                </Box>
            </Tooltip>
        </Box>
    );
}
