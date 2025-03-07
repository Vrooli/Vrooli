import { Button, Popover, useTheme } from "@mui/material";
import { usePopover } from "hooks/usePopover";
import { PopupMenuProps } from "../types.js";

const anchorOrigin = {
    vertical: "bottom",
    horizontal: "center",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "center",
} as const;

export function PopupMenu({
    text = "Menu",
    children,
    ...props
}: PopupMenuProps) {
    const { palette } = useTheme();
    const [anchorEl, handleOpen, handleClose, isOpen] = usePopover();
    const id = isOpen ? "simple-popover" : undefined;

    return (
        <>
            <Button
                aria-describedby={id}
                {...props}
                onClick={handleOpen}
            >
                {text}
            </Button>
            <Popover
                id={id}
                open={isOpen}
                anchorEl={anchorEl}
                onClose={handleClose}
                disableScrollLock={true}
                anchorOrigin={anchorOrigin}
                transformOrigin={transformOrigin}
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
