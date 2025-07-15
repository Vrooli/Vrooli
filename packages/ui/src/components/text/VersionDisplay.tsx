import Box from "@mui/material/Box";
import LinearProgress from "@mui/material/LinearProgress";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import { Tooltip } from "../Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { useCallback, useMemo, useState } from "react";
import { type UsePressEvent, usePress } from "../../hooks/gestures.js";
import { useLocation } from "../../route/router.js";
import { addSearchParams } from "../../route/searchParams.js";
import { PopoverWithArrow } from "../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { type VersionDisplayProps } from "./types.js";

/**
 * Displays version of object.
 * On hover or press, a popup displays a list of all versions. 
 * On click, the version is loaded.
 */
export function VersionDisplay({
    confirmVersionChange,
    currentVersion,
    loading = false,
    prefix = "",
    versions = [],
    ...props
}: VersionDisplayProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const handleVersionChange = useCallback((version: string) => {
        addSearchParams(setLocation, { version });
        window.location.reload();
    }, [setLocation]);

    const openVersion = useCallback((version: string) => {
        if (typeof confirmVersionChange === "function") {
            confirmVersionChange(() => { handleVersionChange(version); });
        } else {
            handleVersionChange(version);
        }
    }, [confirmVersionChange, handleVersionChange]);

    const listItems = useMemo(() => versions?.map((version, index) => {
        return (
            <ListItem
                button
                disabled={version.versionLabel === currentVersion?.versionLabel}
                onClick={() => { openVersion(version.versionLabel); }}
                key={index}
                sx={{
                    padding: "0px 8px",
                }}
            >
                <ListItemText primary={version.versionLabel} />
            </ListItem>
        );
    }), [currentVersion, openVersion, versions]);

    // Versions popup
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = useCallback(({ target }: UsePressEvent) => {
        if (versions.length > 1) {
            setAnchorEl(target as HTMLElement);
        }
    }, [versions.length]);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({
        onLongPress: open,
        onClick: open,
    });

    if (loading && !currentVersion?.versionLabel) return (
        <Box sx={{
            ...(props.sx ?? {}),
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
        }}>
            {prefix && <Typography variant="body1" sx={{ marginRight: "4px" }}>{prefix}</Typography>}
            <LinearProgress
                color="inherit"
                sx={{
                    width: "40px",
                    height: "6px",
                    borderRadius: "12px",
                }}
            />
        </Box>
    );
    return (
        <Box sx={{ ...(props.sx ?? {}) }}>
            {/* Version select popup */}
            <PopoverWithArrow
                anchorEl={anchorEl}
                handleClose={close}
                sxs={{
                    content: {
                        maxHeight: "120px",
                    },
                }}
            >
                {/* Versions list */}
                <List>
                    {listItems}
                </List>
            </PopoverWithArrow>
            {/* Label */}
            <Tooltip title={versions.length > 1 ? "Press to change version" : ""}>
                <Typography
                    {...pressEvents}
                    variant="body2"
                    sx={{
                        cursor: listItems.length > 1 ? "pointer" : "default",
                        color: palette.background.textSecondary,
                    }}
                >{`${prefix}${currentVersion?.versionLabel ?? "x.x.x"}`}</Typography>
            </Tooltip>
        </Box>
    );
}
