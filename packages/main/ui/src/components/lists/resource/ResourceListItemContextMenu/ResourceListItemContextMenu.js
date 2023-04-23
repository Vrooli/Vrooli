import { jsx as _jsx } from "react/jsx-runtime";
import { CopyIcon, DeleteIcon, EditIcon, MoveLeftIcon, MoveLeftRightIcon, MoveRightIcon, ShareIcon } from "@local/icons";
import { getTranslation } from "../../../../utils/display/translationTools";
import { PubSub } from "../../../../utils/pubsub";
import { ListMenu } from "../../../dialogs/ListMenu/ListMenu";
export var ResourceContextMenuOption;
(function (ResourceContextMenuOption) {
    ResourceContextMenuOption["AddBefore"] = "AddBefore";
    ResourceContextMenuOption["AddAfter"] = "AddAfter";
    ResourceContextMenuOption["Copy"] = "Copy";
    ResourceContextMenuOption["Delete"] = "Delete";
    ResourceContextMenuOption["Edit"] = "Edit";
    ResourceContextMenuOption["Move"] = "Move";
    ResourceContextMenuOption["Share"] = "Share";
})(ResourceContextMenuOption || (ResourceContextMenuOption = {}));
const listOptionsMap = {
    [ResourceContextMenuOption.AddBefore]: ["Add resource before", MoveLeftIcon],
    [ResourceContextMenuOption.AddAfter]: ["Add resource after", MoveRightIcon],
    [ResourceContextMenuOption.Copy]: ["Copy link", CopyIcon],
    [ResourceContextMenuOption.Edit]: ["Edit", EditIcon],
    [ResourceContextMenuOption.Delete]: ["Delete", DeleteIcon],
    [ResourceContextMenuOption.Move]: ["Move", MoveLeftRightIcon],
    [ResourceContextMenuOption.Share]: ["Share", ShareIcon],
};
const listOptions = Object.keys(listOptionsMap).map((o) => ({
    label: listOptionsMap[o][0],
    value: o,
    Icon: listOptionsMap[o][1],
}));
export const ResourceListItemContextMenu = ({ canUpdate, id, anchorEl, index, onClose, onAddBefore, onAddAfter, onEdit, onDelete, onMove, resource, zIndex, }) => {
    const onMenuItemSelect = (value) => {
        if (index === null || index < 0)
            return;
        switch (value) {
            case ResourceContextMenuOption.AddBefore:
                onAddBefore(index);
                break;
            case ResourceContextMenuOption.AddAfter:
                onAddAfter(index);
                break;
            case ResourceContextMenuOption.Copy:
                navigator.clipboard.writeText(resource?.link ?? "");
                PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
                break;
            case ResourceContextMenuOption.Delete:
                onDelete(index);
                break;
            case ResourceContextMenuOption.Edit:
                onEdit(index);
                break;
            case ResourceContextMenuOption.Move:
                onMove(index);
                break;
            case ResourceContextMenuOption.Share:
                if (!resource?.link)
                    return;
                const { name, description } = getTranslation(resource, []);
                navigator.share({
                    title: name ?? undefined,
                    text: description ?? undefined,
                    url: resource?.link,
                });
                break;
        }
        onClose();
    };
    const listOptionsFiltered = canUpdate ? listOptions : listOptions.filter(o => o.value === ResourceContextMenuOption.Share);
    return (_jsx(ListMenu, { id: id, anchorEl: anchorEl, data: listOptionsFiltered, onSelect: onMenuItemSelect, onClose: onClose, zIndex: zIndex }));
};
//# sourceMappingURL=ResourceListItemContextMenu.js.map