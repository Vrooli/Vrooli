import { BaseModel } from "./base";

export class TagModel extends BaseModel<any, any> {
    
    constructor(prisma: any) {
        super(prisma, 'tag');
    }

    
}