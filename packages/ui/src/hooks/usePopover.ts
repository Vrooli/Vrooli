import { useState } from "react";

export function usePopover(initialState = null) {
    const [anchorEl, setAnchorEl] = useState<Element | null>(initialState);

    function openPopover(event: Pick<React.MouseEvent<Element>, "currentTarget">, condition = true) {
        if (!condition) return;
        setAnchorEl(event.currentTarget);
    }

    function closePopover() {
        setAnchorEl(null);
    }

    const isPopoverOpen = Boolean(anchorEl);

    return [anchorEl, openPopover, closePopover, isPopoverOpen] as const;
}
