// Abstract class for providing base functionality to models
// (e.g. create, update, delete)
import { CODE } from '@local/shared';
import { PrismaSelect } from '@paljs/plugins';
import { CustomError } from 'error';

export abstract class BaseModel<Input, Model> {
    prisma: any;
    model: string;

    constructor(prisma: any, model: string) {
        this.prisma = prisma;
        this.model = model;
    }

    async findById(id: string, info: any): Promise<Model> {
        if (!Boolean(id)) throw new CustomError(CODE.InvalidArgs);
        return await this.prisma[this.model].findUnique({ where: { id }, ...(new PrismaSelect(info).value) })
    }

    // Create a new model
    abstract create(data: Input): Promise<Model>;
    
    // Update an existing model
    abstract update(id: string, data: Input): Promise<Model>;
    
    // Delete a model
    abstract delete(id: string): Promise<void>;

    // Delete multiple models
    abstract deleteMany(ids: string[]): Promise<number>;
}