import { useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { SessionContext } from "../../../contexts/session.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { type BookmarkListListItemProps } from "../types.js";

export function BookmarkListListItem({
    data,
    onAction,
    ...props
}: BookmarkListListItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    const { title } = useMemo(() => {
        const { title: listTitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: `${listTitle} (${data?.bookmarksCount})`,
        };
    }, [data, session]);

    return (
        <ObjectListItemBase
            {...props}
            titleOverride={title}
            data={data}
            hideUpdateButton={true}
            objectType="BookmarkList"
            onAction={onAction}
            toTheRight={<IconCommon
                decorative
                fill={palette.background.textSecondary}
                name="ChevronRight"
            />}
        />
    );
}
