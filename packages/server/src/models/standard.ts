import { BaseModel } from "./base";

export class StandardModel extends BaseModel<any, any> {
    
    constructor(prisma: any) {
        super(prisma, 'standard');
    }

    
}