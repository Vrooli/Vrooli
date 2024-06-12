import { useState } from "react";

export const usePopover = (initialState = null) => {
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(initialState);

    const openPopover = (event: Pick<React.MouseEvent<HTMLElement>, "currentTarget">, condition = true) => {
        if (!condition) return;
        setAnchorEl(event.currentTarget);
    };

    const closePopover = () => { setAnchorEl(null); };

    const isPopoverOpen = Boolean(anchorEl);

    return [anchorEl, openPopover, closePopover, isPopoverOpen] as const;
};
