import { ListItem, ListItemIcon, ListItemText } from "@mui/material";
import { FormStructureType } from "@vrooli/shared";
import { memo, useCallback } from "react";
import { Icon } from "../../icons/Icons.js";
import type { PopoverListItemProps } from "./FormView.types.js";

export const PopoverListItem = memo(function PopoverListItemMemo({
    iconInfo,
    label,
    onAddHeader,
    onAddInput,
    onAddStructure,
    tag,
    type,
}: Omit<PopoverListItemProps, "key">) {
    const handleClick = useCallback(() => {
        if (type === FormStructureType.Header) {
            if (tag) {
                onAddHeader({ tag });
            } else {
                console.error("Missing tag for header - cannot add header to form.");
            }
        } else if ([FormStructureType.Divider, FormStructureType.Image, FormStructureType.QrCode, FormStructureType.Tip, FormStructureType.Video].includes(type as unknown as FormStructureType)) {
            onAddStructure({ type: type as "Divider" | "Image" | "QrCode" | "Tip" | "Video" });
        } else {
            onAddInput({ type: type as InputType });
        }
    }, [onAddHeader, onAddInput, onAddStructure, tag, type]);

    return (
        <ListItem
            button
            onClick={handleClick}
            data-testid="popover-list-item"
            data-type={type}
            data-tag={tag}
        >
            {iconInfo !== null && iconInfo !== undefined && <ListItemIcon>
                <Icon
                    decorative
                    info={iconInfo}
                    size={24}
                />
            </ListItemIcon>}
            <ListItemText primary={label} />
        </ListItem>
    );
});
