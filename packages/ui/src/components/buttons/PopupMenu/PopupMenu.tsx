import { Button, Popover, useTheme } from "@mui/material";
import { usePopover } from "hooks/usePopover";
import { PopupMenuProps } from "../types";

export const PopupMenu = ({
    text = "Menu",
    children,
    ...props
}: PopupMenuProps) => {
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
};
