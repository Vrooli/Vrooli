import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ChatParticipantListItemProps } from "../types";

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
