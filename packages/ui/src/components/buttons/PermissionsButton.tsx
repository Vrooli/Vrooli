import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import { Tooltip } from "../Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { type TranslationKeyCommon } from "@vrooli/shared";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { usePopover } from "../../hooks/usePopover.js";
import { Button } from "./Button.js";

export type PermissionsFilter = "All" | "Own" | "Public" | "Team";

interface PermissionsMenuProps {
    anchorEl: Element | null;
    onClose: (filter?: PermissionsFilter) => void;
    currentFilter: PermissionsFilter;
}

const PERMISSIONS_OPTIONS: { key: PermissionsFilter; label: TranslationKeyCommon; icon: string }[] = [
    { key: "All", label: "All", icon: "Globe" },
    { key: "Public", label: "Public", icon: "Lock" },
    { key: "Own", label: "MyItems", icon: "Person" },
    { key: "Team", label: "TeamItems", icon: "Group" },
];

const MENU_LIST_PROPS = {
    "aria-label": "Filter search results by permissions",
    "aria-labelledby": "permissions-filter-button",
};

const ICON_STYLE = { marginRight: "8px" };

function PermissionsMenu({
    anchorEl,
    onClose,
    currentFilter,
}: PermissionsMenuProps) {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);

    const handleClose = useCallback(() => {
        onClose();
    }, [onClose]);

    const handleSelect = useCallback((filter: PermissionsFilter) => {
        onClose(filter);
    }, [onClose]);

    const createHandleClick = useCallback((filter: PermissionsFilter) => {
        return () => handleSelect(filter);
    }, [handleSelect]);

    return (
        <Menu
            id="permissions-filter-menu"
            anchorEl={anchorEl}
            disableScrollLock={true}
            open={open}
            onClose={handleClose}
            MenuListProps={MENU_LIST_PROPS}
        >
            {PERMISSIONS_OPTIONS.map(({ key, label, icon }) => (
                <MenuItem
                    key={key}
                    onClick={createHandleClick(key)}
                    selected={currentFilter === key}
                >
                    <IconCommon 
                        decorative 
                        name={icon} 
                        style={ICON_STYLE} 
                    />
                    {t(label)}
                </MenuItem>
            ))}
        </Menu>
    );
}

interface PermissionsButtonProps {
    permissionsFilter: PermissionsFilter;
    setPermissionsFilter: (filter: PermissionsFilter) => void;
}

export function PermissionsButton({
    permissionsFilter,
    setPermissionsFilter,
}: PermissionsButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [anchorEl, openMenu, closeMenu] = usePopover();

    const handleClose = useCallback((filter?: PermissionsFilter) => {
        closeMenu();
        if (filter && filter !== permissionsFilter) {
            setPermissionsFilter(filter);
        }
    }, [closeMenu, permissionsFilter, setPermissionsFilter]);

    const isActive = permissionsFilter !== "All";

    const permissionsLabel = useMemo(() => {
        switch (permissionsFilter) {
            case "Public":
                return t("Public");
            case "Own":
                return t("MyItems");
            case "Team":
                return t("TeamItems");
            default:
                return "";
        }
    }, [permissionsFilter, t]);

    const permissionsIcon = useMemo(() => {
        switch (permissionsFilter) {
            case "Public":
                return "Lock";
            case "Own":
                return "Person";
            case "Team":
                return "Group";
            default:
                return "Shield";
        }
    }, [permissionsFilter]);

    const buttonStyle = useMemo(() => ({
        borderRadius: "24px",
        padding: "2px 12px",
        margin: "2px",
        whiteSpace: "nowrap",
        minHeight: "auto",
        ...(isActive ? {} : {
            borderColor: palette.background.textSecondary,
            color: palette.background.textSecondary,
            borderWidth: "2px",
            borderStyle: "solid",
        }),
    }), [isActive, palette.background.textSecondary]);

    const iconStyle = useMemo(() => ({ 
        marginRight: isActive ? "4px" : "0", 
    }), [isActive]);

    return (
        <>
            <PermissionsMenu
                anchorEl={anchorEl}
                onClose={handleClose}
                currentFilter={permissionsFilter}
            />
            <Tooltip title={t("FilterByPermissions")} placement="top">
                <Button
                    variant={isActive ? "secondary" : "outline"}
                    size="sm"
                    onClick={openMenu}
                    style={buttonStyle}
                    aria-label={t("FilterByPermissions")}
                    id="permissions-filter-button"
                >
                    <IconCommon
                        decorative
                        name={permissionsIcon}
                        style={iconStyle}
                    />
                    {isActive && <Typography variant="caption">{permissionsLabel}</Typography>}
                </Button>
            </Tooltip>
        </>
    );
}
