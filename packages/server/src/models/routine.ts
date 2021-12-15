import { BaseModel } from "./base";

export class RoutineModel extends BaseModel<any, any> {
    
    constructor(prisma: any) {
        super(prisma, 'routine');
    }

    
}