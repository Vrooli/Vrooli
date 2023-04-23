import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { EllipsisIcon } from "@local/icons";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { getActionsDisplayData, getAvailableActions } from "../../../utils/actions/objectActions";
import { getDisplay } from "../../../utils/display/listTools";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { SessionContext } from "../../../utils/SessionContext";
import { ObjectActionDialogs } from "../../dialogs/ObjectActionDialogs/ObjectActionDialogs";
import { ObjectActionMenu } from "../../dialogs/ObjectActionMenu/ObjectActionMenu";
const commonButtonSx = (palette) => ({
    color: "inherit",
    width: "48px",
    height: "100%",
});
const commonIconProps = (palette) => ({
    width: "30px",
    height: "30px",
});
export const ObjectActionsRow = ({ actionData, exclude, object, zIndex, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { actionsDisplayed, actionsExtra } = useMemo(() => {
        const availableActions = getAvailableActions(object, session, exclude);
        let actionsDisplayed;
        let actionsExtra;
        if (availableActions.length > 5) {
            actionsDisplayed = availableActions.slice(0, 4);
            actionsExtra = availableActions.slice(4);
        }
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
    const [anchorEl, setAnchorEl] = useState(null);
    const openOverflowMenu = useCallback((event) => {
        setAnchorEl(event.target);
    }, []);
    const closeOverflowMenu = useCallback(() => setAnchorEl(null), []);
    const actions = useMemo(() => {
        const displayData = getActionsDisplayData(actionsDisplayed);
        const displayedActions = displayData.map((action, index) => {
            const { Icon, iconColor, label, value } = action;
            if (!Icon)
                return null;
            return _jsx(Tooltip, { title: label, children: _jsx(IconButton, { sx: commonButtonSx(palette), onClick: () => { actionData.onActionStart(value); }, children: _jsx(Icon, { ...commonIconProps(palette), fill: iconColor === "default" ? palette.secondary.main : iconColor }) }) }, index);
        });
        if (actionsExtra.length > 0) {
            displayedActions.push(_jsx(Tooltip, { title: "More", children: _jsx(IconButton, { sx: commonButtonSx(palette), onClick: openOverflowMenu, children: _jsx(EllipsisIcon, { ...commonIconProps(palette), fill: palette.secondary.main }) }) }, displayedActions.length));
        }
        return displayedActions;
    }, [actionData, actionsDisplayed, actionsExtra.length, openOverflowMenu, palette]);
    return (_jsxs(Stack, { direction: "row", spacing: 1, sx: {
            marginTop: 1,
            marginBottom: 1,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
        }, children: [_jsx(ObjectActionDialogs, { ...actionData, object: object, zIndex: zIndex + 1 }), actions, actionsExtra.length > 0 && _jsx(ObjectActionMenu, { actionData: actionData, anchorEl: anchorEl, exclude: [...(exclude ?? []), ...actionsDisplayed], object: object, onClose: closeOverflowMenu, zIndex: zIndex + 1 })] }));
};
//# sourceMappingURL=ObjectActionsRow.js.map