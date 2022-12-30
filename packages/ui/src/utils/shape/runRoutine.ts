import { shapeUpdate, shapeUpdatePrims } from "./shapeTools";

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

export const shapeRunRoutineInputUpdate = (o: RunRoutineInputShape,u: RunRoutineInputShape): RunRoutineInputUpdateInput | undefined => {
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'data')
    })
}