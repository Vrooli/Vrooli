import { MeetingInvite, MeetingInviteCreateInput, MeetingInviteUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type MeetingInviteShape = Pick<MeetingInvite, "id" | "message"> & {
    __typename?: "MeetingInvite";
    meeting: { id: string };
    user: { id: string };
}

export const shapeMeetingInvite: ShapeModel<MeetingInviteShape, MeetingInviteCreateInput, MeetingInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message"),
        ...createRel(d, "meeting", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message"),
    }, a),
};
