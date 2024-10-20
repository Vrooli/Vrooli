import { useTheme } from "@mui/material";
import { SessionContext } from "contexts";
import { ChevronRightIcon } from "icons";
import { useContext, useMemo } from "react";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { BookmarkListListItemProps } from "../types";

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
            toTheRight={<ChevronRightIcon fill={palette.background.textSecondary} />}
        />
    );
}
