import { injectable } from 'tsyringe';
import { BaseRepository } from './base.repository.js';
import { Template } from '../schemas/templates.schema.js';

export interface TemplateEntity extends Template {
  id: string;
  created_at: Date;
  updated_at: Date;
}

@injectable()
export class TemplatesRepository extends BaseRepository<TemplateEntity> {
  protected tableName = 'templates';
  protected columns = [
    'id',
    'name',
    'description',
    'category',
    'structure',
    'variables',
    'tags',
    'usage_count',
    'created_at',
    'updated_at'
  ];

  /**
   * Find templates by category
   */
  async findByCategory(category: string): Promise<TemplateEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE category = $1
      ORDER BY usage_count DESC, name ASC
    `;
    
    const result = await this.db.query(query, [category]);
    return result.rows;
  }

  /**
   * Search templates
   */
  async search(searchTerm: string): Promise<TemplateEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE 
        LOWER(name) LIKE LOWER($1) OR
        LOWER(description) LIKE LOWER($1) OR
        $2 = ANY(tags)
      ORDER BY usage_count DESC, name ASC
    `;
    
    const result = await this.db.query(query, [`%${searchTerm}%`, searchTerm]);
    return result.rows;
  }

  /**
   * Get most used templates
   */
  async getMostUsed(limit: number = 10): Promise<TemplateEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      ORDER BY usage_count DESC
      LIMIT $1
    `;
    
    const result = await this.db.query(query, [limit]);
    return result.rows;
  }

  /**
   * Increment usage count
   */
  async incrementUsage(id: string): Promise<void> {
    const query = `
      UPDATE ${this.tableName}
      SET usage_count = usage_count + 1
      WHERE id = $1
    `;
    
    await this.db.query(query, [id]);
  }

  /**
   * Log template application
   */
  async logApplication(
    templateId: string,
    executionId: string,
    context: string,
    variables: Record<string, string>
  ): Promise<void> {
    const query = `
      INSERT INTO template_applications 
      (template_id, execution_id, context, variables, started_at)
      VALUES ($1, $2, $3, $4, NOW())
    `;
    
    await this.db.query(query, [
      templateId,
      executionId,
      context,
      JSON.stringify(variables)
    ]);
  }

  /**
   * Update template application result
   */
  async updateApplicationResult(
    executionId: string,
    status: string,
    results: any[],
    summary: string
  ): Promise<void> {
    const query = `
      UPDATE template_applications
      SET 
        status = $2,
        results = $3,
        summary = $4,
        completed_at = NOW(),
        total_duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000
      WHERE execution_id = $1
    `;
    
    await this.db.query(query, [
      executionId,
      status,
      JSON.stringify(results),
      summary
    ]);
  }
}