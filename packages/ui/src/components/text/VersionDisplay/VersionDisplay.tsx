import { Box, LinearProgress, List, ListItem, ListItemText, Tooltip, Typography } from "@mui/material";
import { VersionDisplayProps } from "../types";
import { useCallback, useMemo, useState } from "react";
import { addSearchParams, usePress } from "utils";
import { PopoverWithArrow } from "components/dialogs";
import { useLocation } from "@shared/route";


/**
 * Displays version of object.
 * On hover or press, a popup displays a list of all versions. 
 * On click, the version is loaded.
 */
export const VersionDisplay = ({
    confirmVersionChange,
    currentVersion,
    loading = false,
    prefix = '',
    versions = [],
    ...props
}: VersionDisplayProps) => {
    const [, setLocation] = useLocation();

    const handleVersionChange = useCallback((version: string) => {
        addSearchParams(setLocation, { version })
        window.location.reload();
    }, [setLocation]);

    const openVersion = useCallback((version: string) => {
        if (typeof confirmVersionChange === 'function') {
            confirmVersionChange(() => { handleVersionChange(version) });
        } else {
            handleVersionChange(version);
        }
    }, [confirmVersionChange, handleVersionChange]);

    const listItems = useMemo(() => versions?.map((version, index) => {
        return (
            <ListItem
                button
                disabled={version === currentVersion}
                onClick={() => { openVersion(version) }}
                key={index}
                sx={{
                    padding: '0px 8px',
                }}
            >
                <ListItemText primary={version} />
            </ListItem>
        )
    }), [currentVersion, openVersion, versions])

    // Versions popup
    const [anchorEl, setAnchorEl] = useState<any | null>(null);
    const open = useCallback((target: EventTarget) => {
        if (versions.length > 1) {
            setAnchorEl(target)
        }
    }, [versions.length]);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({
        onLongPress: open,
        onClick: open,
    });

    if (!currentVersion) return null;
    if (loading) return (
        <Box sx={{
            ...(props.sx ?? {}),
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
        }}>
            {prefix && <Typography variant="body1" sx={{ marginRight: '4px' }}>{prefix}</Typography>}
            <LinearProgress
                color="inherit"
                sx={{
                    width: '40px',
                    height: '6px',
                    borderRadius: '12px'
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
                        maxHeight: '120px',
                    }
                }}
            >
                {/* Versions list */}
                <List>
                    {listItems}
                </List>
            </PopoverWithArrow>
            {/* Label */}
            <Tooltip title={versions.length > 1 ? "Press to change version" : ''}>
                <Typography
                    {...pressEvents}
                    variant="body1"
                    sx={{
                        cursor: 'pointer',
                    }}
                >{`${prefix}${currentVersion}`}</Typography>
            </Tooltip>
        </Box>
    )
}