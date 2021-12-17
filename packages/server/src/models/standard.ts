import { Standard, StandardInput } from "schema/types";
import { BaseModel } from "./base";

export class StandardModel extends BaseModel<StandardInput, Standard> {
    
    constructor(prisma: any) {
        super(prisma, 'standard');
    }

    
}