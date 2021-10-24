import { Request, Response} from 'express';
import pkg from '@prisma/client';
import { PrismaType } from './types';
const { PrismaClient } = pkg;
const prisma = new PrismaClient();

interface ContextProps {
    req: Request;
    res: Response;
}
interface ContextResponse extends ContextProps {
    prisma: PrismaType;
}
export const context = ({ req, res }: ContextProps): ContextResponse => ({
    prisma,
    req,
    res
})