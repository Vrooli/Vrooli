import { UserScheduleFilter, UserScheduleFilterCreateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeTag, TagShape } from "utils";

export type UserScheduleFilterShape = Pick<UserScheduleFilter, 'id' | 'filterType'> & {
    __typename?: 'UserScheduleFilter';
    userSchedule: { id: string },
    tag: TagShape,
}

export const shapeUserScheduleFilter: ShapeModel<UserScheduleFilterShape, UserScheduleFilterCreateInput, null> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'filterType'),
        ...createRel(d, 'userSchedule', ['Connect'], 'one'),
        ...createRel(d, 'tag', ['Create', 'Connect'], 'one', shapeTag),
    })
}