import { SessionContext } from "contexts/SessionContext";
import { useContext, useMemo } from "react";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ChatListItemProps } from "../types";

export function ChatListItem({
    data,
    onAction,
    ...props
}: ChatListItemProps) {
    const session = useContext(SessionContext);

    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);

    return (
        <ObjectListItemBase
            {...props}
            subtitleOverride={subtitle}
            titleOverride={title}
            // TODO
            // toTheRight={
            //     <Stack direction="row" spacing={1}>
            //         {!data?.isRead && <Tooltip title={t("MarkAsRead")}>
            //             <IconButton edge="end" size="small" onClick={() => data && onAction("MarkAsRead", data.id)}>
            //                 <VisibleIcon fill={palette.secondary.main} />
            //             </IconButton>
            //         </Tooltip>}
            //     </Stack>
            // }
            data={data}
            hideUpdateButton={true} // Chats don't have a dedicated update page
            objectType="Chat"
            onAction={onAction}
        />
    );
}
