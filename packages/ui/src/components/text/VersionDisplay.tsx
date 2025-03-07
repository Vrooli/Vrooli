import { Box, LinearProgress, List, ListItem, ListItemText, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { addSearchParams, useLocation } from "route";
import { UsePressEvent, usePress } from "../../hooks/gestures.js";
import { PopoverWithArrow } from "../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { VersionDisplayProps } from "./types.js";

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
