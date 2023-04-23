import { jsx as _jsx } from "react/jsx-runtime";
import { CopyIcon, DeleteIcon, ShareIcon } from "@local/icons";
import { useContext } from "react";
import { getDisplay } from "../../../../utils/display/listTools";
import { getUserLanguages } from "../../../../utils/display/translationTools";
import { getObjectUrl } from "../../../../utils/navigation/openObject";
import { PubSub } from "../../../../utils/pubsub";
import { SessionContext } from "../../../../utils/SessionContext";
import { ListMenu } from "../../../dialogs/ListMenu/ListMenu";
export var DirectoryContextMenuOption;
(function (DirectoryContextMenuOption) {
    DirectoryContextMenuOption["Copy"] = "Copy";
    DirectoryContextMenuOption["Delete"] = "Delete";
    DirectoryContextMenuOption["Share"] = "Share";
})(DirectoryContextMenuOption || (DirectoryContextMenuOption = {}));
const listOptionsMap = {
    [DirectoryContextMenuOption.Copy]: ["Copy link", CopyIcon],
    [DirectoryContextMenuOption.Delete]: ["Delete", DeleteIcon],
    [DirectoryContextMenuOption.Share]: ["Share", ShareIcon],
};
const listOptions = Object.keys(listOptionsMap).map((o) => ({
    label: listOptionsMap[o][0],
    value: o,
    Icon: listOptionsMap[o][1],
}));
export const DirectoryListItemContextMenu = ({ canUpdate, data, id, anchorEl, index, onClose, onDelete, zIndex, }) => {
    const session = useContext(SessionContext);
    const onMenuItemSelect = (value) => {
        if (index === null || index < 0)
            return;
        switch (value) {
            case DirectoryContextMenuOption.Copy:
                navigator.clipboard.writeText(getObjectUrl(data) ?? "");
                PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
                break;
            case DirectoryContextMenuOption.Delete:
                onDelete(index);
                break;
            case DirectoryContextMenuOption.Share:
                if (!data)
                    return;
                const { title, subtitle } = getDisplay(data, getUserLanguages(session));
                navigator.share({
                    title: title ?? undefined,
                    text: subtitle ?? undefined,
                    url: getObjectUrl(data) ?? undefined,
                });
                break;
        }
        onClose();
    };
    const listOptionsFiltered = canUpdate ? listOptions : listOptions.filter(o => o.value === DirectoryContextMenuOption.Share);
    return (_jsx(ListMenu, { id: id, anchorEl: anchorEl, data: listOptionsFiltered, onSelect: onMenuItemSelect, onClose: onClose, zIndex: zIndex }));
};
//# sourceMappingURL=DirectoryListItemContextMenu.js.map