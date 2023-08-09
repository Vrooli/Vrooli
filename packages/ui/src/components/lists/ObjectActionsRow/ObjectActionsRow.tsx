import { IconButton, Palette, Stack, Tooltip, useTheme } from "@mui/material";
import { ObjectActionDialogs } from "components/dialogs/ObjectActionDialogs/ObjectActionDialogs";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { EllipsisIcon } from "icons";
import React, { useCallback, useContext, useMemo, useState } from "react";
import { getActionsDisplayData, getAvailableActions, ObjectAction } from "utils/actions/objectActions";
import { getDisplay, ListObject } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { SessionContext } from "utils/SessionContext";
import { ObjectActionsRowProps } from "../types";

const commonButtonSx = (palette: Palette) => ({
    color: "inherit",
    width: "48px",
    height: "100%",
});

const commonIconProps = (palette: Palette) => ({
    width: "30px",
    height: "30px",
});

/**
 * Horizontal list of action icons displayed on an object's view page. 
 * Available icons are same as ObjectActionMenu. Actions that are not available are hidden.
 * If there are more than 5 actions, rest are hidden in an overflow menu (i.e. ObjectActionMenu).
 */
export const ObjectActionsRow = <T extends ListObject>({
    actionData,
    exclude,
    object,
    zIndex,
}: ObjectActionsRowProps<T>) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    const { actionsDisplayed, actionsExtra } = useMemo(() => {
        const availableActions = getAvailableActions(object, session, exclude);
        let actionsDisplayed: ObjectAction[];
        let actionsExtra: ObjectAction[];
        // If there are more than 5 actions, display the first 4 in the row, and the rest in the overflow menu
        if (availableActions.length > 5) {
            actionsDisplayed = availableActions.slice(0, 4);
            actionsExtra = availableActions.slice(4);
        }
        // If there are 5 or less actions, display them all in the row
        else {
            actionsDisplayed = availableActions;
            actionsExtra = [];
        }
        return {
            actionsDisplayed,
            actionsExtra,
            id: object?.id,
            name: getDisplay(object, getUserLanguages(session)).title,
            objectType: object?.__typename,
        };
    }, [exclude, object, session]);

    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const openOverflowMenu = useCallback((event: React.MouseEvent) => {
        setAnchorEl(event.target as HTMLElement);
    }, []);
    const closeOverflowMenu = useCallback(() => setAnchorEl(null), []);

    const actions = useMemo(() => {
        const displayData = getActionsDisplayData(actionsDisplayed);
        const displayedActions = displayData.map((action, index) => {
            const { Icon, iconColor, label, value } = action;
            if (!Icon) return null;
            return <Tooltip title={label} key={index}>
                <IconButton sx={commonButtonSx(palette)} onClick={() => { actionData.onActionStart(value); }}>
                    <Icon {...commonIconProps(palette)} fill={iconColor === "default" ? palette.secondary.main : iconColor} />
                </IconButton>
            </Tooltip>;
        });
        // If there are extra actions, display an ellipsis button
        if (actionsExtra.length > 0) {
            displayedActions.push(
                <Tooltip title="More" key={displayedActions.length}>
                    <IconButton sx={commonButtonSx(palette)} onClick={openOverflowMenu}>
                        <EllipsisIcon {...commonIconProps(palette)} fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>,
            );
        }
        return displayedActions;
    }, [actionData, actionsDisplayed, actionsExtra.length, openOverflowMenu, palette]);

    return (
        <Stack
            direction="row"
            spacing={1}
            sx={{
                marginTop: 1,
                marginBottom: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
            }}
        >
            {/* Action dialogs */}
            <ObjectActionDialogs
                {...actionData}
                object={object}
                zIndex={zIndex + 1}
            />
            {/* Displayed actions */}
            {actions}
            {/* Overflow menu */}
            {actionsExtra.length > 0 && <ObjectActionMenu
                actionData={actionData}
                anchorEl={anchorEl}
                exclude={[...(exclude ?? []), ...actionsDisplayed]}
                object={object}
                onClose={closeOverflowMenu}
                zIndex={zIndex + 1}
            />}
        </Stack>
    );
};
