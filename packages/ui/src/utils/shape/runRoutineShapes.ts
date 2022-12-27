import { shapeUpdate } from "./shapeTools";

export type RunRoutineInputShape = {
    id: RunRoutineInput['id'];
    data: RunRoutineInput['data'];
    input: {
        id: RunRoutineInput['input']['id'];
    }
}

export const shapeRunRoutineInputCreate = (item: RunRoutineInputShape): RunRoutineInputCreateInput => ({
    id: item.id,
    data: item.data,
    inputId: item.input.id,
})

export const shapeRunRoutineInputUpdate = (
    original: RunRoutineInputShape,
    updated: RunRoutineInputShape
): RunRoutineInputUpdateInput | undefined => {
    if (updated.data === original.data) return undefined;
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        data: u.data
    }), 'id')
}