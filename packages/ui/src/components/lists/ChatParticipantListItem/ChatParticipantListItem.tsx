import { DateDisplay } from "components/text/DateDisplay.js";
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
