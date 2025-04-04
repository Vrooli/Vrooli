import { Button, Popover, styled } from "@mui/material";
import { usePopover } from "../../../hooks/usePopover.js";
import { PopupMenuProps } from "../types.js";

const anchorOrigin = {
    vertical: "bottom",
    horizontal: "center",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "center",
} as const;

const StyledPopover = styled(Popover)(({ theme }) => ({
    "& .MuiPopover-paper": {
        background: theme.palette.background.paper,
        borderRadius: theme.spacing(2),
    },
}));

export function PopupMenu({
    text = "Menu",
    children,
    ...props
}: PopupMenuProps) {
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
            <StyledPopover
                id={id}
                open={isOpen}
                anchorEl={anchorEl}
                onClose={handleClose}
                disableScrollLock={true}
                anchorOrigin={anchorOrigin}
                transformOrigin={transformOrigin}
            >
                {children}
            </StyledPopover>
        </>
    );
}
