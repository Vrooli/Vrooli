import { ChatParticipant, ChatParticipantUpdateInput, User } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate, updatePrims } from "./tools";

export type ChatParticipantShape = Pick<ChatParticipant, "id"> & {
    __typename: "ChatParticipant";
    user: Pick<User, "__typename" | "updated_at" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}

export const shapeChatParticipant: ShapeModel<ChatParticipantShape, null, ChatParticipantUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
    }),
};
