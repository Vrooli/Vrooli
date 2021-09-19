import knex from 'knex';
import knexfile from './knexfile.js';

const environment = process.env.NODE_ENV || 'development';
export const db = knex(knexfile[environment]);