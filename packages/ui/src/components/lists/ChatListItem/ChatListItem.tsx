import { ChatParticipant } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { displayDate } from "utils/display/stringTools";
import { SessionContext } from "utils/SessionContext";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ChatListItemProps } from "../types";

export function ChatListItem({
    data,
    onAction,
    ...props
}: ChatListItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    console.log("in ChatListItem", data, props);

    const { title, subtitle } = useMemo(() => {
        if (!data) return { title: "Chat", subtitle: "" };
        const { title, subtitle } = getDisplay(data);
        // If title and subtitle exist, return them
        if (title && title.length && subtitle && subtitle.length) {
            return { title, subtitle };
        }
        // If there is only one participant besides the current user, return their name
        const otherParticipants: ChatParticipant[] = data.participants?.filter(p => p.user?.id !== getCurrentUser(session)?.id) ?? [] as ChatParticipant[];
        if (otherParticipants.length === 1) {
            const participantDisplay = getDisplay(otherParticipants[0]);
            return {
                title: `Chat with ${participantDisplay.title}`,
                subtitle: "",
            };
        }
        // If there are multiple participants, return the "Group Chat" title
        if (otherParticipants.length > 1) {
            return {
                title: `Group chat (${data.participantsCount - 1})`,
                subtitle: "",
            };
        }
        // Otherwise, return "Chat" + date
        return {
            title: `Chat ${displayDate(data.created_at)}`,
            subtitle: "",
        };
    }, [data, session]);

    return (
        <ObjectListItemBase
            {...props}
            subtitleOverride={subtitle}
            titleOverride={title}
            toTheRight={
                <Stack direction="row" spacing={1}>
                    {/* {!data?.isRead && <Tooltip title={t("MarkAsRead")}>
                        <IconButton edge="end" size="small" onClick={() => data && onAction("MarkAsRead", data.id)}>
                            <VisibleIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip>} */}
                    {/* <Tooltip title={t("Delete")}>
                        <IconButton edge="end" size="small" onClick={() => data && onAction("Delete", data.id)}>
                            <DeleteIcon fill={palette.secondary.main} />
                        </IconButton>
                    </Tooltip> */}
                </Stack>
            }
            data={data}
            objectType="Chat"
        />
    );
}
