import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useMemo } from "react";
import { getActionsDisplayData } from "../../../utils/actions/objectActions";
import { ListMenu } from "../ListMenu/ListMenu";
import { ObjectActionDialogs } from "../ObjectActionDialogs/ObjectActionDialogs";
export const ObjectActionMenu = ({ actionData, anchorEl, exclude, object, onClose, zIndex, }) => {
    const displayedActions = useMemo(() => getActionsDisplayData(actionData.availableActions.filter(action => !exclude?.includes(action))), [actionData.availableActions, exclude]);
    const onSelect = useCallback((action) => {
        actionData.onActionStart(action);
        onClose();
    }, [actionData, onClose]);
    return (_jsxs(_Fragment, { children: [_jsx(ObjectActionDialogs, { ...actionData, object: object, zIndex: zIndex + 1 }), _jsx(ListMenu, { anchorEl: anchorEl, data: displayedActions, id: `${object?.__typename}-options-menu-${object?.id}`, onClose: onClose, onSelect: onSelect, zIndex: zIndex })] }));
};
//# sourceMappingURL=ObjectActionMenu.js.map