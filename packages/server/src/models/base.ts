// Abstract class for providing base functionality to models
// (e.g. create, update, delete)
import { CODE } from '@local/shared';
import { PrismaSelect } from '@paljs/plugins';
import { CustomError } from '../error';

interface InputInterface {
    id?: string;
}

export abstract class BaseModel<Input extends InputInterface, Model> {
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
    async create(data: any, info: any): Promise<any> {
        throw new Error("Method not implemented.");
    }
    
    // Update an existing model
    async update(data: Input, info: any): Promise<Model> {
        if (!Boolean(data.id)) throw new CustomError(CODE.InvalidArgs);
        return await this.prisma[this.model].update({ where: { id: data.id }, data, ...(new PrismaSelect(info).value) })
    }
    
    // Delete a model
    async delete(id: string): Promise<void> {
        await this.prisma[this.model].delete({ where: { id } });
    }

    // Delete multiple models
    async deleteMany(ids: string[]): Promise<number> {
        return await this.prisma[this.model].deleteMany({ whereIn: { id: ids } });
    }
}