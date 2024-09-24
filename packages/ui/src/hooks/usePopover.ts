import { useState } from "react";

export function usePopover(initialState = null) {
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(initialState);

    function openPopover(event: Pick<React.MouseEvent<HTMLElement>, "currentTarget">, condition = true) {
        if (!condition) return;
        setAnchorEl(event.currentTarget);
    }

    function closePopover() {
        setAnchorEl(null);
    }

    const isPopoverOpen = Boolean(anchorEl);

    return [anchorEl, openPopover, closePopover, isPopoverOpen] as const;
}
