import { PrismaSelect } from "@paljs/plugins";
import { BaseModel } from "./base";
import pkg from '@prisma/client';
const { ResourceFor } = pkg;

export class ResourceModel extends BaseModel<any, any> {
    
    constructor(prisma: any) {
        super(prisma, 'resource');
    }

    /**
     * Creates a resource object
     * @param data 
     * @param info 
     * @returns 
     */
    async create(data: any, info: any) {
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, forId, ...resourceData } = data;
        // Create base object
        const resource = await this.prisma.resouce.create({ data: resourceData });
        // Create join object
        await this.applyToObject(resource, createdFor, forId);
        // Return query
        return await this.prisma.resource.findOne({ where: { id: resource.id }, ...(new PrismaSelect(info).value) });
    }

    /**
     * Applies a resource object to an actor, project, organization, or routine
     * @param resource 
     * @returns
     */
    async applyToObject(resource: any, createdFor: typeof ResourceFor, forId: string) {
        // Maps routine apply types to the correct prisma join tables
        const applyMap = {
            [ResourceFor.ORGANIZATION]: 'organizationResources',
            [ResourceFor.PROJECT]: 'projectResources',
            [ResourceFor.ROUTINE_CONTEXTUAL]: 'routineResourcesContextual',
            [ResourceFor.ROUTINE_DONATION]: 'routineResourcesDonation',
            [ResourceFor.ROUTINE_EXTERNAL]: 'routineResourcesExternal',
            [ResourceFor.USER]: 'userResources'
        }
        return await this.prisma[applyMap[createdFor]].create({ data: {
            forId,
            resourceId: resource.id
        }})
    }
}