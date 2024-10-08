import { MeetingInvite, MeetingInviteCreateInput, MeetingInviteUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { MeetingShape } from "./meeting";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type MeetingInviteShape = Pick<MeetingInvite, "id" | "message"> & {
    __typename: "MeetingInvite";
    meeting: CanConnect<MeetingShape>;
    user: { id: string };
}

export const shapeMeetingInvite: ShapeModel<MeetingInviteShape, MeetingInviteCreateInput, MeetingInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message"),
        ...createRel(d, "meeting", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message"),
    }),
};
