import { Request, Response} from 'express';
import pkg from '@prisma/client';
import { PrismaType } from './types';
import { createDriver } from './db/driver';
import { Driver } from 'neo4j-driver';
const { PrismaClient } = pkg;
const prisma = new PrismaClient();

interface ContextProps {
    req: Request;
    res: Response;
}
export interface Context extends ContextProps {
    prisma: PrismaType;
    driver: Driver;
}
export const context = ({ req, res }: ContextProps): Context => ({
    prisma, //TODO remove prisma
    driver: createDriver(),
    req,
    res,
})