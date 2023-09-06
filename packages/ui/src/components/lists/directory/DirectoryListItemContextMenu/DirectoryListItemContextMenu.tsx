import { ListMenu } from "components/dialogs/ListMenu/ListMenu";
import { ListMenuItemData } from "components/dialogs/types";
import { SessionContext } from "contexts/SessionContext";
import { CopyIcon, DeleteIcon, ShareIcon } from "icons";
import { useContext } from "react";
import { SvgComponent } from "types";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { DirectoryListItemContextMenuProps } from "../types";

export enum DirectoryContextMenuOption {
    Copy = "Copy",
    Delete = "Delete",
    Share = "Share",
}

const listOptionsMap: { [key in DirectoryContextMenuOption]: [string, SvgComponent] } = {
    [DirectoryContextMenuOption.Copy]: ["Copy link", CopyIcon],
    [DirectoryContextMenuOption.Delete]: ["Delete", DeleteIcon],
    [DirectoryContextMenuOption.Share]: ["Share", ShareIcon],
};

const listOptions: ListMenuItemData<DirectoryContextMenuOption>[] = Object.keys(listOptionsMap).map((o) => ({
    label: listOptionsMap[o][0],
    value: o as DirectoryContextMenuOption,
    Icon: listOptionsMap[o][1],
}));

// Custom context menu for nodes
export const DirectoryListItemContextMenu = ({
    canUpdate,
    data,
    id,
    anchorEl,
    index,
    onClose,
    onDelete,
}: DirectoryListItemContextMenuProps) => {
    const session = useContext(SessionContext);

    const onMenuItemSelect = (value: DirectoryContextMenuOption) => {
        if (index === null || index < 0) return;
        switch (value) {
            case DirectoryContextMenuOption.Copy:
                navigator.clipboard.writeText(getObjectUrl(data as any) ?? "");
                PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
                break;
            case DirectoryContextMenuOption.Delete:
                onDelete(index);
                break;
            case DirectoryContextMenuOption.Share: {
                if (!data) return;
                const { title, subtitle } = getDisplay(data as any, getUserLanguages(session));
                navigator.share({
                    title: title ?? undefined,
                    text: subtitle ?? undefined,
                    url: getObjectUrl(data as any) ?? undefined,
                });
                break;
            }
        }
        onClose();
    };

    const listOptionsFiltered = canUpdate ? listOptions : listOptions.filter(o => o.value === DirectoryContextMenuOption.Share);

    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            data={listOptionsFiltered}
            onSelect={onMenuItemSelect}
            onClose={onClose}
        />
    );
};
