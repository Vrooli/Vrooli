import Box from "@mui/material/Box";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Popover from "@mui/material/Popover";
import React, { useCallback, useState, type MouseEvent } from "react";
import { Icon, IconCommon, IconRoutine } from "../../../../icons/Icons.js";
import type { ExternalApp } from "../utils.js";

// Add supported external apps here
const externalApps: ExternalApp[] = [
];

const popoverAnchorOrigin = { vertical: "top", horizontal: "left" } as const;
const popoverTransformOrigin = { vertical: "bottom", horizontal: "left" } as const;

/**
 * PlusMenu Component - renders the popover for additional actions.
 */
interface PlusMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
    onAttachFile?: () => unknown;
    onConnectExternalApp?: () => unknown;
    onTakePhoto?: () => unknown;
    onAddRoutine?: () => unknown;
}

export const PlusMenu: React.FC<PlusMenuProps> = React.memo(
    ({
        anchorEl,
        onClose,
        onAttachFile,
        onConnectExternalApp,
        onTakePhoto,
        onAddRoutine,
    }) => {
        const [externalAppAnchor, setExternalAppAnchor] = useState<HTMLElement | null>(null);

        function handleOpenExternalApps(event: MouseEvent<HTMLElement>) {
            setExternalAppAnchor(event.currentTarget);
        }

        function handleCloseExternalApps() {
            setExternalAppAnchor(null);
        }

        const handleAppConnection = useCallback((appId: string) => {
            // Toggle connection for this app, e.g., call a function like toggleAppConnection(app.id)
            handleCloseExternalApps();
        }, []);

        return (
            <>
                <Popover
                    open={Boolean(anchorEl)}
                    anchorEl={anchorEl}
                    onClose={onClose}
                    anchorOrigin={popoverAnchorOrigin}
                    transformOrigin={popoverTransformOrigin}
                >
                    <Box>
                        {onAttachFile && <MenuItem onClick={onAttachFile}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="File"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Attach File" secondary="Attach a file from your device" />
                        </MenuItem>}
                        {externalApps.length > 0 && onConnectExternalApp && <MenuItem onClick={handleOpenExternalApps}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="Link"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Connect External App" secondary="Connect an external app to your account" />
                        </MenuItem>}
                        {onTakePhoto && <MenuItem onClick={onTakePhoto}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="CameraOpen"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Take Photo" secondary="Take a photo from your device" />
                        </MenuItem>}
                        {onAddRoutine && <MenuItem onClick={onAddRoutine}>
                            <ListItemIcon>
                                <IconRoutine
                                    decorative
                                    name="Routine"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Add Routine" secondary="Allow the AI to perform actions" />
                        </MenuItem>}
                    </Box>
                </Popover>
                <Menu
                    anchorEl={externalAppAnchor}
                    open={Boolean(externalAppAnchor)}
                    onClose={handleCloseExternalApps}
                    anchorOrigin={popoverAnchorOrigin}
                    transformOrigin={popoverTransformOrigin}
                >
                    {externalApps.map((app) => {
                        function connectApp() {
                            handleAppConnection(app.id);
                        }

                        return (
                            <MenuItem
                                key={app.id}
                                onClick={connectApp}
                            >
                                <ListItemIcon>
                                    <Icon decorative info={app.iconInfo} />
                                </ListItemIcon>
                                <ListItemText primary={app.name} secondary="Description about the app" />
                                {app.connected && <IconCommon
                                    decorative
                                    name="Complete"
                                />}
                            </MenuItem>
                        );
                    })}
                </Menu>
            </>
        );
    });
PlusMenu.displayName = "PlusMenu";
