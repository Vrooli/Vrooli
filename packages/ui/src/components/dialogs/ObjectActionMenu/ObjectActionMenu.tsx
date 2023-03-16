import { useCallback, useMemo } from "react";
import { getActionsDisplayData, ObjectAction } from "utils/actions/objectActions";
import { ListMenu } from "../ListMenu/ListMenu";
import { ObjectActionDialogs } from "../ObjectActionDialogs/ObjectActionDialogs";
import { ObjectActionMenuProps } from "../types";

export const ObjectActionMenu = ({
    actionData,
    anchorEl,
    exclude,
    object,
    onClose,
    session,
    zIndex,
}: ObjectActionMenuProps) => {

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
                session={session}
                zIndex={zIndex + 1}
            />
            {/* The menu to select an action */}
            <ListMenu
                anchorEl={anchorEl}
                data={displayedActions}
                id={`${object?.__typename}-options-menu-${object?.id}`}
                onClose={onClose}
                onSelect={onSelect}
                zIndex={zIndex}
            />
        </>
    )
}