import { IconButton, List, ListItem, ListItemIcon, ListItemText, Menu, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { useZIndex } from "hooks/useZIndex";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
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
}: ListMenuProps<T>) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const open = Boolean(anchorEl);
    const zIndex = useZIndex(open);

    const items = useMemo(() => data?.map(({
        label,
        labelKey,
        value,
        Icon,
        iconColor,
        helpData,
    }, index) => {
        const itemText = <ListItemText primary={labelKey ? t(labelKey) : label} />;
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
    }), [data, id, onClose, onSelect, palette.background.textSecondary, t]);

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
            onClose={(_e, reason: "backdropClick" | "escapeKeyDown" | "tabKeyDown") => {
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
            />}
            <List>
                {items}
            </List>
        </Menu>
    );
}
