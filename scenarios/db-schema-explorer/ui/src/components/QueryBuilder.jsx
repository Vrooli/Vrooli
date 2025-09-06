import React, { useState } from 'react'
import {
  Box,
  TextField,
  Button,
  Paper,
  Typography,
  CircularProgress,
  Alert,
  Divider,
  Chip,
  IconButton,
  Tooltip,
  Grid
} from '@mui/material'
import {
  PlayArrow as RunIcon,
  AutoAwesome as AIIcon,
  ContentCopy as CopyIcon,
  Speed as OptimizeIcon,
  Save as SaveIcon
} from '@mui/icons-material'
import Editor from 'react-simple-code-editor'
import { highlight, languages } from 'prismjs/components/prism-core'
import 'prismjs/components/prism-sql'
import { useMutation } from 'react-query'
import axios from 'axios'

const API_BASE = '/api/v1'

function QueryBuilder({ database, onNotification }) {
  const [naturalLanguage, setNaturalLanguage] = useState('')
  const [generatedSQL, setGeneratedSQL] = useState('')
  const [queryResults, setQueryResults] = useState(null)
  const [explanation, setExplanation] = useState('')
  const [confidence, setConfidence] = useState(0)

  // Generate SQL from natural language
  const generateMutation = useMutation(
    (prompt) => axios.post(`${API_BASE}/query/generate`, {
      natural_language: prompt,
      database_context: database,
      include_explanation: true
    }),
    {
      onSuccess: (response) => {
        const data = response.data
        setGeneratedSQL(data.sql || '')
        setExplanation(data.explanation || '')
        setConfidence(data.confidence || 0)
        onNotification('SQL generated successfully', 'success')
      },
      onError: (error) => {
        onNotification('Failed to generate SQL', 'error')
      }
    }
  )

  // Execute SQL query
  const executeMutation = useMutation(
    (sql) => axios.post(`${API_BASE}/query/execute`, {
      sql: sql,
      database_name: database,
      limit: 100
    }),
    {
      onSuccess: (response) => {
        setQueryResults(response.data)
        onNotification(`Query executed in ${response.data.execution_time_ms}ms`, 'success')
      },
      onError: (error) => {
        onNotification('Query execution failed', 'error')
      }
    }
  )

  // Optimize SQL query
  const optimizeMutation = useMutation(
    (sql) => axios.post(`${API_BASE}/query/optimize`, {
      sql: sql,
      database_name: database
    }),
    {
      onSuccess: (response) => {
        if (response.data.optimized_sql) {
          setGeneratedSQL(response.data.optimized_sql)
          onNotification('Query optimized', 'success')
        }
      },
      onError: (error) => {
        onNotification('Optimization failed', 'error')
      }
    }
  )

  const handleGenerate = () => {
    if (naturalLanguage.trim()) {
      generateMutation.mutate(naturalLanguage)
    }
  }

  const handleExecute = () => {
    if (generatedSQL.trim()) {
      executeMutation.mutate(generatedSQL)
    }
  }

  const handleOptimize = () => {
    if (generatedSQL.trim()) {
      optimizeMutation.mutate(generatedSQL)
    }
  }

  const handleCopySQL = () => {
    navigator.clipboard.writeText(generatedSQL)
    onNotification('SQL copied to clipboard', 'info')
  }

  const isLoading = generateMutation.isLoading || executeMutation.isLoading || optimizeMutation.isLoading

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 2 }}>
      {/* Natural Language Input */}
      <Paper sx={{ p: 2 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Natural Language Query
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <TextField
            fullWidth
            variant="outlined"
            placeholder="e.g., Show me all tables with more than 10 columns"
            value={naturalLanguage}
            onChange={(e) => setNaturalLanguage(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleGenerate()}
            disabled={isLoading}
          />
          <Button
            variant="contained"
            startIcon={<AIIcon />}
            onClick={handleGenerate}
            disabled={isLoading || !naturalLanguage.trim()}
          >
            Generate
          </Button>
        </Box>
        
        {explanation && (
          <Alert severity="info" sx={{ mt: 2 }}>
            {explanation}
          </Alert>
        )}
        
        {confidence > 0 && (
          <Box sx={{ mt: 1 }}>
            <Chip
              label={`Confidence: ${confidence}%`}
              color={confidence > 80 ? 'success' : confidence > 60 ? 'warning' : 'error'}
              size="small"
            />
          </Box>
        )}
      </Paper>

      {/* SQL Editor */}
      <Paper sx={{ p: 2, flex: 1, minHeight: 200 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">SQL Query</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Tooltip title="Copy SQL">
              <IconButton onClick={handleCopySQL} disabled={!generatedSQL}>
                <CopyIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title="Optimize Query">
              <IconButton onClick={handleOptimize} disabled={!generatedSQL || isLoading}>
                <OptimizeIcon />
              </IconButton>
            </Tooltip>
            <Button
              variant="contained"
              color="primary"
              startIcon={<RunIcon />}
              onClick={handleExecute}
              disabled={!generatedSQL || isLoading}
            >
              Execute
            </Button>
          </Box>
        </Box>
        
        <Box
          sx={{
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1,
            p: 2,
            bgcolor: 'background.default',
            fontFamily: 'monospace',
            fontSize: 14,
            minHeight: 150,
          }}
        >
          <Editor
            value={generatedSQL}
            onValueChange={setGeneratedSQL}
            highlight={(code) => highlight(code, languages.sql)}
            padding={0}
            style={{
              fontFamily: '"Fira Code", monospace',
              fontSize: 14,
              color: '#00ff88',
            }}
          />
        </Box>
      </Paper>

      {/* Query Results */}
      {queryResults && (
        <Paper sx={{ p: 2, maxHeight: 400, overflow: 'auto' }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Results</Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Chip label={`${queryResults.row_count} rows`} size="small" />
              <Chip label={`${queryResults.execution_time_ms}ms`} size="small" color="success" />
            </Box>
          </Box>
          
          {queryResults.rows && queryResults.rows.length > 0 ? (
            <Box sx={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    {queryResults.columns.map((col, idx) => (
                      <th
                        key={idx}
                        style={{
                          padding: '8px',
                          borderBottom: '2px solid #444',
                          textAlign: 'left',
                          color: '#00ff88',
                        }}
                      >
                        {col}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {queryResults.rows.map((row, rowIdx) => (
                    <tr key={rowIdx}>
                      {row.map((cell, cellIdx) => (
                        <td
                          key={cellIdx}
                          style={{
                            padding: '8px',
                            borderBottom: '1px solid #333',
                            fontFamily: 'monospace',
                            fontSize: '13px',
                          }}
                        >
                          {cell === null ? 'NULL' : String(cell)}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </Box>
          ) : (
            <Typography color="text.secondary">No results returned</Typography>
          )}
        </Paper>
      )}

      {/* Loading Overlay */}
      {isLoading && (
        <Box
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            bgcolor: 'rgba(0,0,0,0.7)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 1000,
          }}
        >
          <CircularProgress />
        </Box>
      )}
    </Box>
  )
}

export default QueryBuilder