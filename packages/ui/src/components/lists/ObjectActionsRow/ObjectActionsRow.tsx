import { ListObject } from "@local/shared";
import { Box, IconButton, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { ObjectActionDialogs } from "components/dialogs/ObjectActionDialogs/ObjectActionDialogs.js";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { EllipsisIcon } from "icons/common.js";
import React, { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ObjectAction, getActionsDisplayData, getAvailableActions } from "utils/actions/objectActions.js";
import { getDisplay } from "utils/display/listTools.js";
import { getUserLanguages } from "utils/display/translationTools.js";
import { SessionContext } from "../../../contexts.js";
import { ObjectActionsRowProps } from "../types.js";

const MAX_ACTIONS_BEFORE_OVERFLOW = 5;

function commonIconProps() {
    return {
        width: "24px",
        height: "24px",
    };
}

const OuterBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    alignItems: "center",
    justifyContent: "space-between",
}));

const ActionIconButton = styled(IconButton)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    width: "32px",
    height: "100%",
    padding: 0,
}));

const ActionIconWithLabelBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    alignItems: "center",
    cursor: "pointer",
    color: theme.palette.background.textSecondary,
}));

/**
 * Horizontal list of action icons displayed on an object's view page. 
 * Available icons are same as ObjectActionMenu. Actions that are not available are hidden.
 * If there are more than 5 actions, rest are hidden in an overflow menu (i.e. ObjectActionMenu).
 */
export function ObjectActionsRow<T extends ListObject>({
    actionData,
    exclude,
    object,
}: ObjectActionsRowProps<T>) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { actionsDisplayed, actionsExtra } = useMemo(() => {
        const availableActions = getAvailableActions(object, session, exclude);
        let actionsDisplayed: ObjectAction[];
        let actionsExtra: ObjectAction[];
        // If there are more than 5 actions, display the first 4 in the row, and the rest in the overflow menu
        if (availableActions.length > MAX_ACTIONS_BEFORE_OVERFLOW) {
            actionsDisplayed = availableActions.slice(0, MAX_ACTIONS_BEFORE_OVERFLOW - 1);
            actionsExtra = availableActions.slice(MAX_ACTIONS_BEFORE_OVERFLOW - 1);
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
            const { Icon, iconColor, labelKey, value } = action;
            if (!Icon) return null;

            function handleClick() {
                actionData.onActionStart(value);
            }

            return (
                <ActionIconWithLabelBox key={index} onClick={handleClick}>
                    <Icon {...commonIconProps()} fill={iconColor === "default" ? "currentColor" : iconColor} />
                    <Typography variant="body2">
                        {labelKey && t(labelKey, { count: 1 })}
                    </Typography>
                </ActionIconWithLabelBox>
            );
        });
        // If there are extra actions, display an ellipsis button
        if (actionsExtra.length > 0) {
            displayedActions.push(
                <Tooltip title="More" key={displayedActions.length}>
                    <ActionIconButton onClick={openOverflowMenu}>
                        <EllipsisIcon {...commonIconProps()} fill={palette.secondary.main} />
                    </ActionIconButton>
                </Tooltip>,
            );
        }
        return displayedActions;
    }, [actionData, actionsDisplayed, actionsExtra.length, openOverflowMenu, palette, t]);

    const overflowMenuExclude = useMemo(function overflowMenuExcludeMemo() {
        return [...(exclude ?? []), ...actionsDisplayed];
    }, [exclude, actionsDisplayed]);

    return (
        <OuterBox>
            <ObjectActionDialogs {...actionData} object={object} />
            {actions}
            {/* Overflow menu */}
            {actionsExtra.length > 0 && <ObjectActionMenu
                actionData={actionData}
                anchorEl={anchorEl}
                exclude={overflowMenuExclude}
                object={object}
                onClose={closeOverflowMenu}
            />}
        </OuterBox>
    );
}
