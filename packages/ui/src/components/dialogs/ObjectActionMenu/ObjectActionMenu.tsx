import { useCallback, useMemo } from "react";
import { type ObjectAction, getActionsDisplayData } from "../../../utils/actions/objectActions.js";
import { ListMenu } from "../ListMenu/ListMenu.js";
import { ObjectActionDialogs } from "../ObjectActionDialogs/ObjectActionDialogs.js";
import { type ObjectActionMenuProps } from "../types.js";

export function ObjectActionMenu({
    actionData,
    anchorEl,
    exclude,
    object,
    onClose,
}: ObjectActionMenuProps) {
    const displayedActions = useMemo(() => getActionsDisplayData(actionData.availableActions.filter(action => !exclude?.includes(action))), [actionData.availableActions, exclude]);

    const onSelect = useCallback((action: ObjectAction) => {
        actionData.onActionStart(action);
        onClose();
    }, [actionData, onClose]);

    return (
        <>
            {/* Action dialogs */}
            <ObjectActionDialogs
                {...actionData}
                object={object}
            />
            {/* The menu to select an action */}
            <ListMenu
                anchorEl={anchorEl}
                data={displayedActions}
                id={`${object?.__typename}-options-menu-${object?.id}`}
                onClose={onClose}
                onSelect={onSelect}
            />
        </>
    );
}
