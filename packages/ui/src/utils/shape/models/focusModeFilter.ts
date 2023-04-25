import { FocusModeFilter, FocusModeFilterCreateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeTag, TagShape } from "./tag";
import { createPrims, createRel } from "./tools";

export type FocusModeFilterShape = Pick<FocusModeFilter, 'id' | 'filterType'> & {
    __typename?: 'FocusModeFilter';
    focusMode: { id: string },
    tag: TagShape,
}

export const shapeFocusModeFilter: ShapeModel<FocusModeFilterShape, FocusModeFilterCreateInput, null> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'filterType'),
        ...createRel(d, 'focusMode', ['Connect'], 'one'),
        ...createRel(d, 'tag', ['Create', 'Connect'], 'one', shapeTag),
    })
}