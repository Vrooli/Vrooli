// Get Issues API Endpoint
// Query and filter issues with pagination

import { pgClient } from '@windmill/common';

export async function main(
  apiToken?: string,
  appIds?: string[],
  statuses?: string[],
  priorities?: string[],
  types?: string[],
  search?: string,
  patternGroupId?: string,
  assignedAgentId?: string,
  reporterId?: string,
  tags?: string[],
  dateFrom?: string,
  dateTo?: string,
  sortBy: 'created_at' | 'updated_at' | 'priority' | 'status' = 'created_at',
  sortOrder: 'asc' | 'desc' = 'desc',
  page: number = 1,
  pageSize: number = 20,
  includeRelated: boolean = false,
  includeComments: boolean = false
): Promise<{
  success: boolean;
  issues: any[];
  totalCount: number;
  page: number;
  pageSize: number;
  totalPages: number;
}> {
  const client = await pgClient();
  
  try {
    // Build WHERE clause dynamically
    const conditions: string[] = [];
    const params: any[] = [];
    let paramIndex = 1;
    
    // If API token provided, filter by app
    if (apiToken) {
      const appResult = await client.query(
        'SELECT id FROM apps WHERE api_token = $1',
        [apiToken]
      );
      if (appResult.rows.length > 0) {
        conditions.push(`i.app_id = $${paramIndex}`);
        params.push(appResult.rows[0].id);
        paramIndex++;
      }
    }
    
    // Filter by app IDs
    if (appIds && appIds.length > 0) {
      conditions.push(`i.app_id = ANY($${paramIndex})`);
      params.push(appIds);
      paramIndex++;
    }
    
    // Filter by statuses
    if (statuses && statuses.length > 0) {
      conditions.push(`i.status = ANY($${paramIndex})`);
      params.push(statuses);
      paramIndex++;
    }
    
    // Filter by priorities
    if (priorities && priorities.length > 0) {
      conditions.push(`i.priority = ANY($${paramIndex})`);
      params.push(priorities);
      paramIndex++;
    }
    
    // Filter by types
    if (types && types.length > 0) {
      conditions.push(`i.type = ANY($${paramIndex})`);
      params.push(types);
      paramIndex++;
    }
    
    // Search in title and description
    if (search) {
      conditions.push(`(i.title ILIKE $${paramIndex} OR i.description ILIKE $${paramIndex})`);
      params.push(`%${search}%`);
      paramIndex++;
    }
    
    // Filter by pattern group
    if (patternGroupId) {
      conditions.push(`i.pattern_group_id = $${paramIndex}`);
      params.push(patternGroupId);
      paramIndex++;
    }
    
    // Filter by assigned agent
    if (assignedAgentId) {
      conditions.push(`i.assigned_agent_id = $${paramIndex}`);
      params.push(assignedAgentId);
      paramIndex++;
    }
    
    // Filter by reporter
    if (reporterId) {
      conditions.push(`i.reporter_id = $${paramIndex}`);
      params.push(reporterId);
      paramIndex++;
    }
    
    // Filter by tags
    if (tags && tags.length > 0) {
      conditions.push(`i.tags && $${paramIndex}`);
      params.push(tags);
      paramIndex++;
    }
    
    // Date range filters
    if (dateFrom) {
      conditions.push(`i.created_at >= $${paramIndex}`);
      params.push(dateFrom);
      paramIndex++;
    }
    
    if (dateTo) {
      conditions.push(`i.created_at <= $${paramIndex}`);
      params.push(dateTo);
      paramIndex++;
    }
    
    const whereClause = conditions.length > 0 ? `WHERE ${conditions.join(' AND ')}` : '';
    
    // Build ORDER BY clause
    let orderColumn = 'i.created_at';
    if (sortBy === 'updated_at') orderColumn = 'i.updated_at';
    else if (sortBy === 'priority') {
      orderColumn = `CASE i.priority 
        WHEN 'critical' THEN 1 
        WHEN 'high' THEN 2 
        WHEN 'medium' THEN 3 
        WHEN 'low' THEN 4 
      END`;
    } else if (sortBy === 'status') {
      orderColumn = `CASE i.status 
        WHEN 'open' THEN 1 
        WHEN 'investigating' THEN 2 
        WHEN 'in_progress' THEN 3 
        WHEN 'fixed' THEN 4 
        WHEN 'closed' THEN 5 
        WHEN 'wont_fix' THEN 6 
      END`;
    }
    
    const orderClause = `ORDER BY ${orderColumn} ${sortOrder}`;
    
    // Get total count
    const countQuery = `
      SELECT COUNT(*) as total 
      FROM issues i 
      ${whereClause}
    `;
    const countResult = await client.query(countQuery, params);
    const totalCount = parseInt(countResult.rows[0].total);
    
    // Calculate pagination
    const offset = (page - 1) * pageSize;
    const totalPages = Math.ceil(totalCount / pageSize);
    
    // Get issues with pagination
    const issuesQuery = `
      SELECT 
        i.*,
        a.display_name as app_name,
        a.type as app_type,
        ag.display_name as assigned_agent_name,
        pg.name as pattern_group_name,
        CASE 
          WHEN i.resolved_at IS NOT NULL 
          THEN EXTRACT(EPOCH FROM (i.resolved_at - i.created_at))/3600 
          ELSE NULL 
        END as resolution_hours
      FROM issues i
      JOIN apps a ON i.app_id = a.id
      LEFT JOIN agents ag ON i.assigned_agent_id = ag.id
      LEFT JOIN pattern_groups pg ON i.pattern_group_id = pg.id
      ${whereClause}
      ${orderClause}
      LIMIT $${paramIndex} OFFSET $${paramIndex + 1}
    `;
    
    params.push(pageSize, offset);
    const issuesResult = await client.query(issuesQuery, params);
    
    let issues = issuesResult.rows;
    
    // Optionally include related issues
    if (includeRelated && issues.length > 0) {
      const issueIds = issues.map(i => i.id);
      const relatedQuery = `
        SELECT 
          r.*,
          i2.title as related_title,
          i2.status as related_status,
          a2.display_name as related_app_name
        FROM related_issues r
        JOIN issues i2 ON r.related_issue_id = i2.id
        JOIN apps a2 ON i2.app_id = a2.id
        WHERE r.issue_id = ANY($1)
      `;
      const relatedResult = await client.query(relatedQuery, [issueIds]);
      
      // Group related issues by issue ID
      const relatedByIssue = relatedResult.rows.reduce((acc, rel) => {
        if (!acc[rel.issue_id]) acc[rel.issue_id] = [];
        acc[rel.issue_id].push(rel);
        return acc;
      }, {} as Record<string, any[]>);
      
      // Attach related issues to each issue
      issues = issues.map(issue => ({
        ...issue,
        related_issues: relatedByIssue[issue.id] || []
      }));
    }
    
    // Optionally include comments
    if (includeComments && issues.length > 0) {
      const issueIds = issues.map(i => i.id);
      const commentsQuery = `
        SELECT * FROM issue_comments 
        WHERE issue_id = ANY($1)
        ORDER BY created_at DESC
      `;
      const commentsResult = await client.query(commentsQuery, [issueIds]);
      
      // Group comments by issue ID
      const commentsByIssue = commentsResult.rows.reduce((acc, comment) => {
        if (!acc[comment.issue_id]) acc[comment.issue_id] = [];
        acc[comment.issue_id].push(comment);
        return acc;
      }, {} as Record<string, any[]>);
      
      // Attach comments to each issue
      issues = issues.map(issue => ({
        ...issue,
        comments: commentsByIssue[issue.id] || []
      }));
    }
    
    return {
      success: true,
      issues,
      totalCount,
      page,
      pageSize,
      totalPages
    };
    
  } catch (error) {
    console.error('Error fetching issues:', error);
    return {
      success: false,
      issues: [],
      totalCount: 0,
      page: 1,
      pageSize,
      totalPages: 0
    };
  } finally {
    await client.end();
  }
}