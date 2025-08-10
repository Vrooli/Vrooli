import { Pool, PoolClient } from 'pg';
import { inject } from 'tsyringe';
import { logger } from '../utils/logger.js';
import { NotFoundError } from '../utils/errors.js';

export interface BaseEntity {
  id: string;
  created_at: Date;
  updated_at: Date;
}

export abstract class BaseRepository<T extends BaseEntity> {
  protected abstract tableName: string;
  protected abstract columns: string[];

  constructor(
    @inject('DbPool') protected db: Pool
  ) {}

  /**
   * Find entity by ID
   */
  async findById(id: string): Promise<T | null> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE id = $1
    `;
    
    const result = await this.db.query(query, [id]);
    return result.rows[0] || null;
  }

  /**
   * Find entity by ID or throw error
   */
  async findByIdOrThrow(id: string): Promise<T> {
    const entity = await this.findById(id);
    if (!entity) {
      throw new NotFoundError(this.tableName, id);
    }
    return entity;
  }

  /**
   * Find all entities with optional filters
   */
  async findAll(filters: Record<string, any> = {}): Promise<T[]> {
    const whereConditions: string[] = [];
    const values: any[] = [];
    let paramCounter = 1;

    for (const [key, value] of Object.entries(filters)) {
      if (value !== undefined && value !== null) {
        whereConditions.push(`${key} = $${paramCounter}`);
        values.push(value);
        paramCounter++;
      }
    }

    const whereClause = whereConditions.length > 0
      ? `WHERE ${whereConditions.join(' AND ')}`
      : '';

    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      ${whereClause}
      ORDER BY created_at DESC
    `;

    const result = await this.db.query(query, values);
    return result.rows;
  }

  /**
   * Find entities with pagination
   */
  async findPaginated(
    limit: number = 20,
    offset: number = 0,
    filters: Record<string, any> = {}
  ): Promise<{ data: T[]; total: number }> {
    const whereConditions: string[] = [];
    const values: any[] = [];
    let paramCounter = 1;

    for (const [key, value] of Object.entries(filters)) {
      if (value !== undefined && value !== null) {
        whereConditions.push(`${key} = $${paramCounter}`);
        values.push(value);
        paramCounter++;
      }
    }

    const whereClause = whereConditions.length > 0
      ? `WHERE ${whereConditions.join(' AND ')}`
      : '';

    // Get total count
    const countQuery = `
      SELECT COUNT(*) as total
      FROM ${this.tableName}
      ${whereClause}
    `;
    const countResult = await this.db.query(countQuery, values);
    const total = parseInt(countResult.rows[0].total);

    // Get paginated data
    values.push(limit, offset);
    const dataQuery = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      ${whereClause}
      ORDER BY created_at DESC
      LIMIT $${paramCounter} OFFSET $${paramCounter + 1}
    `;

    const dataResult = await this.db.query(dataQuery, values);

    return {
      data: dataResult.rows,
      total
    };
  }

  /**
   * Create a new entity
   */
  async create(data: Partial<T>): Promise<T> {
    const keys = Object.keys(data);
    const values = Object.values(data);
    const placeholders = keys.map((_, index) => `$${index + 1}`);

    const query = `
      INSERT INTO ${this.tableName} (${keys.join(', ')})
      VALUES (${placeholders.join(', ')})
      RETURNING ${this.columns.join(', ')}
    `;

    const result = await this.db.query(query, values);
    return result.rows[0];
  }

  /**
   * Update an entity
   */
  async update(id: string, data: Partial<T>): Promise<T> {
    const keys = Object.keys(data);
    const values = Object.values(data);
    const setClause = keys.map((key, index) => `${key} = $${index + 2}`).join(', ');

    const query = `
      UPDATE ${this.tableName}
      SET ${setClause}, updated_at = NOW()
      WHERE id = $1
      RETURNING ${this.columns.join(', ')}
    `;

    const result = await this.db.query(query, [id, ...values]);
    
    if (result.rows.length === 0) {
      throw new NotFoundError(this.tableName, id);
    }

    return result.rows[0];
  }

  /**
   * Delete an entity
   */
  async delete(id: string): Promise<boolean> {
    const query = `
      DELETE FROM ${this.tableName}
      WHERE id = $1
      RETURNING id
    `;

    const result = await this.db.query(query, [id]);
    return (result.rowCount ?? 0) > 0;
  }

  /**
   * Execute a transaction
   */
  async transaction<R>(
    callback: (client: PoolClient) => Promise<R>
  ): Promise<R> {
    const client = await this.db.connect();
    
    try {
      await client.query('BEGIN');
      const result = await callback(client);
      await client.query('COMMIT');
      return result;
    } catch (error) {
      await client.query('ROLLBACK');
      logger.error('Transaction rolled back:', error);
      throw error;
    } finally {
      client.release();
    }
  }
}