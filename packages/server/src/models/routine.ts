import { Routine, RoutineInput } from "schema/types";
import { BaseModel } from "./base";

export class RoutineModel extends BaseModel<RoutineInput, Routine> {
    
    constructor(prisma: any) {
        super(prisma, 'routine');
    }

    
}