import { UserScheduleFilter, UserScheduleFilterCreateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeTag, TagShape } from "./tag";
import { createPrims, createRel } from "./tools";

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