import { Tag, TagInput } from "schema/types";
import { BaseModel } from "./base";

export class TagModel extends BaseModel<TagInput, Tag> {
    
    constructor(prisma: any) {
        super(prisma, 'tag');
    }

    
}