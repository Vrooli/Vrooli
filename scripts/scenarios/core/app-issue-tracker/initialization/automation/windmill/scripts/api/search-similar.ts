// Search Similar Issues API Endpoint
// Semantic search for finding similar or duplicate issues

import { pgClient, httpRequest } from '@windmill/common';

export async function main(
  query: string,
  appId?: string,
  threshold: number = 0.7,
  limit: number = 10,
  includeResolved: boolean = false
): Promise<{
  success: boolean;
  results: Array<{
    id: string;
    title: string;
    description: string;
    app_name: string;
    status: string;
    priority: string;
    similarity: number;
    resolution?: string;
  }>;
  patterns?: Array<{
    pattern_group: string;
    count: number;
    common_issues: string[];
  }>;
}> {
  const client = await pgClient();
  
  try {
    // Generate embedding for search query
    const ollamaResponse = await httpRequest({
      url: 'http://ollama:11434/api/embeddings',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        model: 'nomic-embed-text',
        prompt: query
      })
    });
    
    if (!ollamaResponse.data.embedding) {
      return {
        success: false,
        results: [],
        patterns: []
      };
    }
    
    const embedding = ollamaResponse.data.embedding;
    
    // Search in Qdrant for similar issues
    const qdrantResponse = await httpRequest({
      url: 'http://qdrant:6333/collections/issues/points/search',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        vector: embedding,
        limit: limit * 2, // Get more to filter later
        with_payload: true,
        score_threshold: threshold
      })
    });
    
    const vectorResults = qdrantResponse.data.result || [];
    
    // Get full issue details from PostgreSQL
    if (vectorResults.length === 0) {
      return {
        success: true,
        results: [],
        patterns: []
      };
    }
    
    const issueIds = vectorResults.map(r => r.id);
    
    // Build query with filters
    let conditions = ['i.id = ANY($1)'];
    const params: any[] = [issueIds];
    let paramIndex = 2;
    
    if (appId) {
      conditions.push(`i.app_id = $${paramIndex}`);
      params.push(appId);
      paramIndex++;
    }
    
    if (!includeResolved) {
      conditions.push(`i.status NOT IN ('closed', 'fixed', 'wont_fix')`);
    }
    
    const whereClause = conditions.join(' AND ');
    
    const issuesQuery = `
      SELECT 
        i.id,
        i.title,
        i.description,
        i.status,
        i.priority,
        i.type,
        i.root_cause,
        i.suggested_fix,
        i.pattern_group_id,
        a.display_name as app_name,
        pg.name as pattern_group_name
      FROM issues i
      JOIN apps a ON i.app_id = a.id
      LEFT JOIN pattern_groups pg ON i.pattern_group_id = pg.id
      WHERE ${whereClause}
    `;
    
    const issuesResult = await client.query(issuesQuery, params);
    
    // Combine with similarity scores
    const results = issuesResult.rows.map(issue => {
      const vectorMatch = vectorResults.find(v => v.id === issue.id);
      return {
        ...issue,
        similarity: vectorMatch ? vectorMatch.score : 0,
        resolution: issue.status === 'fixed' ? issue.suggested_fix : undefined
      };
    }).sort((a, b) => b.similarity - a.similarity).slice(0, limit);
    
    // Analyze patterns if multiple similar issues found
    let patterns = [];
    if (results.length >= 3) {
      // Group by pattern
      const patternGroups = results.reduce((acc, issue) => {
        if (issue.pattern_group_id) {
          if (!acc[issue.pattern_group_id]) {
            acc[issue.pattern_group_id] = {
              pattern_group: issue.pattern_group_name,
              count: 0,
              common_issues: []
            };
          }
          acc[issue.pattern_group_id].count++;
          acc[issue.pattern_group_id].common_issues.push(issue.title);
        }
        return acc;
      }, {} as Record<string, any>);
      
      patterns = Object.values(patternGroups);
      
      // Check for emergent patterns (issues without pattern group but similar)
      const unpatterned = results.filter(r => !r.pattern_group_id);
      if (unpatterned.length >= 3) {
        // Extract common keywords
        const allText = unpatterned.map(r => `${r.title} ${r.description}`).join(' ').toLowerCase();
        const words = allText.split(/\W+/).filter(w => w.length > 3);
        const wordCount = words.reduce((acc, word) => {
          acc[word] = (acc[word] || 0) + 1;
          return acc;
        }, {} as Record<string, number>);
        
        const commonWords = Object.entries(wordCount)
          .filter(([_, count]) => count >= unpatterned.length * 0.7)
          .map(([word]) => word)
          .slice(0, 5);
        
        if (commonWords.length > 0) {
          patterns.push({
            pattern_group: 'Emerging Pattern',
            count: unpatterned.length,
            common_issues: unpatterned.slice(0, 3).map(r => r.title),
            keywords: commonWords
          });
          
          // Consider creating a new pattern group
          await suggestNewPattern(client, commonWords, unpatterned);
        }
      }
    }
    
    return {
      success: true,
      results: results.map(r => ({
        id: r.id,
        title: r.title,
        description: r.description,
        app_name: r.app_name,
        status: r.status,
        priority: r.priority,
        similarity: Math.round(r.similarity * 100) / 100,
        resolution: r.resolution
      })),
      patterns
    };
    
  } catch (error) {
    console.error('Error searching similar issues:', error);
    return {
      success: false,
      results: [],
      patterns: []
    };
  } finally {
    await client.end();
  }
}

async function suggestNewPattern(
  client: any,
  keywords: string[],
  issues: any[]
): Promise<void> {
  try {
    // Check if pattern suggestion already exists
    const existingCheck = await client.query(
      'SELECT id FROM pattern_groups WHERE common_keywords && $1',
      [keywords]
    );
    
    if (existingCheck.rows.length === 0) {
      // Create pattern suggestion
      await client.query(`
        INSERT INTO pattern_groups (
          name,
          description,
          pattern_type,
          common_keywords,
          total_issues,
          active_issues
        ) VALUES (
          $1, $2, $3, $4, $5, $6
        )
      `, [
        `Suggested: ${keywords.slice(0, 3).join('-')}`,
        `Automatically detected pattern based on ${issues.length} similar issues`,
        'error_pattern',
        keywords,
        issues.length,
        issues.filter(i => ['open', 'investigating', 'in_progress'].includes(i.status)).length
      ]);
      
      console.log(`Created pattern suggestion for keywords: ${keywords.join(', ')}`);
    }
  } catch (error) {
    console.error('Error creating pattern suggestion:', error);
  }
}