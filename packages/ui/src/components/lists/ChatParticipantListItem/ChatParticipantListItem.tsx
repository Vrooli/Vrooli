import { DateDisplay } from "../../text/DateDisplay.js";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { ChatParticipantListItemProps } from "../types.js";

export function ChatParticipantListItem({
    data,
    ...props
}: ChatParticipantListItemProps) {
    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                data?.createdAt ?
                    <DateDisplay
                        showIcon={true}
                        textBeforeDate="Joined"
                        timestamp={data?.createdAt}
                    /> : null
            }
            data={data}
            objectType="ChatParticipant"
        />
    );
}
