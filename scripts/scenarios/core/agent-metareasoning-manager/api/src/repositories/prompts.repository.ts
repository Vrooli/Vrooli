import { injectable } from 'tsyringe';
import { BaseRepository } from './base.repository.js';
import { Prompt } from '../schemas/prompts.schema.js';

export interface PromptEntity extends Prompt {
  id: string;
  created_at: Date;
  updated_at: Date;
}

@injectable()
export class PromptsRepository extends BaseRepository<PromptEntity> {
  protected tableName = 'prompts';
  protected columns = [
    'id',
    'name',
    'description',
    'category',
    'template',
    'variables',
    'tags',
    'status',
    'version',
    'created_at',
    'updated_at'
  ];

  /**
   * Find prompts by category
   */
  async findByCategory(category: string): Promise<PromptEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE category = $1 AND status = 'active'
      ORDER BY name ASC
    `;
    
    const result = await this.db.query(query, [category]);
    return result.rows;
  }

  /**
   * Find prompts by tag
   */
  async findByTag(tag: string): Promise<PromptEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE $1 = ANY(tags) AND status = 'active'
      ORDER BY name ASC
    `;
    
    const result = await this.db.query(query, [tag]);
    return result.rows;
  }

  /**
   * Search prompts by name or description
   */
  async search(searchTerm: string): Promise<PromptEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE (
        LOWER(name) LIKE LOWER($1) OR
        LOWER(description) LIKE LOWER($1)
      ) AND status = 'active'
      ORDER BY name ASC
    `;
    
    const result = await this.db.query(query, [`%${searchTerm}%`]);
    return result.rows;
  }

  /**
   * Increment version number for a prompt
   */
  async incrementVersion(id: string): Promise<PromptEntity> {
    const query = `
      UPDATE ${this.tableName}
      SET version = version + 1, updated_at = NOW()
      WHERE id = $1
      RETURNING ${this.columns.join(', ')}
    `;
    
    const result = await this.db.query(query, [id]);
    return result.rows[0];
  }

  /**
   * Get prompt history (versions)
   */
  async getHistory(id: string): Promise<any[]> {
    const query = `
      SELECT * FROM prompt_history
      WHERE prompt_id = $1
      ORDER BY version DESC
    `;
    
    const result = await this.db.query(query, [id]);
    return result.rows;
  }
}