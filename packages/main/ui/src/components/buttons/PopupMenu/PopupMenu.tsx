import { Button, Popover, useTheme } from "@mui/material";
import { useState } from "react";
import { PopupMenuProps } from "../types";

export function PopupMenu({
    text = "Menu",
    children,
    ...props
}: PopupMenuProps) {
    const { palette } = useTheme();

    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

    const handleClick = (event: any) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    const id = open ? "simple-popover" : undefined;
    return (
        <>
            <Button aria-describedby={id} {...props} onClick={handleClick}>
                {text}
            </Button>
            <Popover
                id={id}
                open={open}
                anchorEl={anchorEl}
                onClose={handleClose}
                disableScrollLock={true}
                anchorOrigin={{
                    vertical: "bottom",
                    horizontal: "center",
                }}
                transformOrigin={{
                    vertical: "top",
                    horizontal: "center",
                }}
                sx={{
                    "& .MuiPopover-paper": {
                        background: palette.primary.light,
                        borderRadius: "24px",
                    },
                }}
            >
                {children}
            </Popover>
        </>
    );
}
