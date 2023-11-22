import { useTheme } from "@mui/material";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ChatParticipantListItemProps } from "../types";

export function ChatParticipantListItem({
    data,
    ...props
}: ChatParticipantListItemProps) {
    const { palette } = useTheme();

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                data?.created_at ?
                    <DateDisplay
                        showIcon={true}
                        textBeforeDate="Joined"
                        timestamp={data?.created_at}
                    /> : null
            }
            data={data}
            objectType="ChatParticipant"
        />
    );
}
