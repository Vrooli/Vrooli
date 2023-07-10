import { IconButton, List, ListItem, ListItemIcon, ListItemText, Menu, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { useMemo } from "react";
import { MenuTitle } from "../MenuTitle/MenuTitle";
import { ListMenuProps } from "../types";

const titleId = "list-menu-title";

export function ListMenu<T>({
    id,
    anchorEl,
    onSelect,
    onClose,
    title,
    data,
    zIndex,
}: ListMenuProps<T>) {
    const { palette } = useTheme();

    const open = Boolean(anchorEl);

    const items = useMemo(() => data?.map(({ label, value, Icon, iconColor, preview, helpData }, index) => {
        const itemText = <ListItemText primary={label} secondary={preview ? "Coming Soon" : null} sx={{
            // Style secondary
            "& .MuiListItemText-secondary": {
                color: "red",
            },
        }} />;
        const fill = !iconColor || ["default", "unset"].includes(iconColor) ? palette.background.textSecondary : iconColor;
        const itemIcon = Icon ? (
            <ListItemIcon>
                <Icon fill={fill} />
            </ListItemIcon>
        ) : null;
        const helpIcon = helpData ? (
            <IconButton edge="end" onClick={(e) => e.stopPropagation()} >
                <HelpButton {...helpData} />
            </IconButton>
        ) : null;

        // Handle Tab and Shift+Tab manually
        const handleKeyDown = (event) => {
            if (event.key === "Tab") {
                event.preventDefault(); // Stop the default behavior
                const direction = event.shiftKey ? -1 : 1;
                const nextElement = document.getElementById(`${id}-list-item-${index + direction}`);
                console.log("key down?", nextElement, (index + direction), event);
                nextElement && nextElement.focus();
            }
        };

        return (
            <ListItem
                disabled={preview}
                button
                onClick={() => { onSelect(value); onClose(); }}
                onKeyDown={handleKeyDown}
                key={`list-item-${index}`}
                id={`${id}-list-item-${index}`}
                tabIndex={index}
            >
                {itemIcon}
                {itemText}
                {helpIcon}
            </ListItem>
        );
    }), [data, id, onClose, onSelect, palette.background.textSecondary]);

    return (
        <Menu
            id={id}
            disableScrollLock={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: "bottom",
                horizontal: "center",
            }}
            transformOrigin={{
                vertical: "top",
                horizontal: "center",
            }}
            onClose={(e, reason: any) => {
                if (reason !== "tabKeyDown") {
                    onClose();
                }
            }}
            tabIndex={-1}
            sx={{
                zIndex,
                "& .MuiMenu-paper": {
                    background: palette.background.default,
                },
                "& .MuiMenu-list": {
                    paddingTop: "0",
                },
            }}
        >
            {title && <MenuTitle
                ariaLabel={titleId}
                title={title}
                onClose={() => { onClose(); }}
                zIndex={zIndex}
            />}
            <List>
                {items}
            </List>
        </Menu>
    );
}
