import { IconButton } from "../../buttons/IconButton.js";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Menu from "@mui/material/Menu";
import { useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Icon } from "../../../icons/Icons.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { HelpButton } from "../../buttons/HelpButton.js";
import { MenuTitle } from "../MenuTitle/MenuTitle.js";
import { type ListMenuProps } from "../types.js";

const titleId = "list-menu-title";

function stopPropagation(event: React.MouseEvent) {
    event.stopPropagation();
}

const anchorOrigin = {
    vertical: "bottom",
    horizontal: "center",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "center",
} as const;

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

    const items = useMemo(() => data?.map(({
        label,
        labelKey,
        value,
        iconColor,
        iconInfo,
        helpData,
    }, index) => {
        const itemText = <ListItemText primary={labelKey ? t(labelKey, { count: 1 }) : label} />;
        const fill = !iconColor || ["default", "unset"].includes(iconColor) ? palette.background.textSecondary : iconColor;
        const itemIcon = iconInfo ? (
            <ListItemIcon>
                <Icon fill={fill} info={iconInfo} />
            </ListItemIcon>
        ) : null;
        const helpIcon = helpData ? (
            <IconButton edge="end" onClick={stopPropagation} variant="transparent">
                <HelpButton {...helpData} />
            </IconButton>
        ) : null;

        // Handle Tab and Shift+Tab manually
        function handleKeyDown(event: React.KeyboardEvent) {
            if (event.key === "Tab") {
                event.preventDefault(); // Stop the default behavior
                const direction = event.shiftKey ? -1 : 1;
                const nextElement = document.getElementById(`${id}-list-item-${index + direction}`);
                nextElement && nextElement.focus();
            }
        }

        function handleClick() {
            onSelect(value);
            onClose();
        }

        return (
            <ListItem
                button
                onClick={handleClick}
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

    const handleClose = useCallback(function handleCloseCallback(_event: unknown, reason: "backdropClick" | "escapeKeyDown" | "tabKeyDown") {
        if (reason !== "tabKeyDown") {
            onClose();
        }
    }, [onClose]);

    return (
        <Menu
            id={id}
            disableScrollLock={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={anchorOrigin}
            transformOrigin={transformOrigin}
            onClose={handleClose}
            tabIndex={-1}
            sx={{
                zIndex: Z_INDEX.Popup,
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
