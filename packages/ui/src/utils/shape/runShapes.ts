import { RunInputCreateInput, RunInputUpdateInput } from "graphql/generated/globalTypes";
import { RunInput } from "types";
import { shapeUpdate } from "./shapeTools";

export type RunInputShape = {
    id: RunInput['id'];
    data: RunInput['data'];
    input: {
        id: RunInput['input']['id'];
    }
}

export const shapeRunInputCreate = (item: RunInputShape): RunInputCreateInput => ({
    id: item.id,
    data: item.data,
    inputId: item.input.id,
})

export const shapeRunInputUpdate = (
    original: RunInputShape,
    updated: RunInputShape
): RunInputUpdateInput | undefined => {
    if (updated.data === original.data) return undefined;
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        data: u.data
    }), 'id')
}