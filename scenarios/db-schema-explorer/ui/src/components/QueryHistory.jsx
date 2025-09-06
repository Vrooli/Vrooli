import React, { useState } from 'react'
import {
  Box,
  Paper,
  Typography,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Chip,
  TextField,
  InputAdornment,
  Tooltip,
  Divider,
  Button
} from '@mui/material'
import {
  Search as SearchIcon,
  ContentCopy as CopyIcon,
  PlayArrow as RunIcon,
  ThumbUp as ThumbUpIcon,
  ThumbDown as ThumbDownIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon
} from '@mui/icons-material'
import { useQuery, useMutation } from 'react-query'
import axios from 'axios'

const API_BASE = '/api/v1'

function QueryHistory({ database, onNotification }) {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedQuery, setSelectedQuery] = useState(null)

  // Fetch query history
  const { data: historyData, isLoading, refetch } = useQuery(
    ['queryHistory', database],
    () => axios.get(`${API_BASE}/query/history`, {
      params: { database }
    }).then(res => res.data),
    {
      refetchInterval: 30000, // Refresh every 30 seconds
    }
  )

  const handleCopySQL = (sql) => {
    navigator.clipboard.writeText(sql)
    onNotification('SQL copied to clipboard', 'info')
  }

  const handleRunQuery = (sql) => {
    // This would trigger the query execution
    onNotification('Query execution triggered', 'info')
  }

  const handleFeedback = (queryId, feedback) => {
    // This would update the feedback for the query
    onNotification(`Feedback recorded: ${feedback}`, 'success')
  }

  const formatDate = (dateString) => {
    const date = new Date(dateString)
    return date.toLocaleString()
  }

  const getQueryTypeColor = (type) => {
    switch (type?.toUpperCase()) {
      case 'SELECT': return 'primary'
      case 'INSERT': return 'success'
      case 'UPDATE': return 'warning'
      case 'DELETE': return 'error'
      default: return 'default'
    }
  }

  const filteredHistory = historyData?.history?.filter(item => {
    if (!searchTerm) return true
    const term = searchTerm.toLowerCase()
    return (
      item.natural_language?.toLowerCase().includes(term) ||
      item.sql?.toLowerCase().includes(term) ||
      item.query_type?.toLowerCase().includes(term)
    )
  }) || []

  return (
    <Box sx={{ height: '100%', display: 'flex', gap: 2 }}>
      {/* History List */}
      <Paper sx={{ flex: 1, p: 2, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">Query History</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Chip label={`${filteredHistory.length} queries`} size="small" />
            <IconButton onClick={() => refetch()} size="small">
              <RefreshIcon />
            </IconButton>
          </Box>
        </Box>

        <TextField
          fullWidth
          size="small"
          placeholder="Search queries..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ mb: 2 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />

        <List sx={{ flex: 1, overflow: 'auto' }}>
          {filteredHistory.map((item, index) => (
            <React.Fragment key={item.id || index}>
              <ListItem
                button
                selected={selectedQuery?.id === item.id}
                onClick={() => setSelectedQuery(item)}
                sx={{
                  flexDirection: 'column',
                  alignItems: 'flex-start',
                  py: 2,
                  '&:hover': { bgcolor: 'action.hover' }
                }}
              >
                <Box sx={{ width: '100%', display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                    {item.natural_language}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {formatDate(item.created_at)}
                  </Typography>
                </Box>
                
                <Box sx={{ width: '100%', mb: 1 }}>
                  <Typography
                    variant="caption"
                    sx={{
                      fontFamily: 'monospace',
                      color: 'primary.main',
                      display: 'block',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {item.sql}
                  </Typography>
                </Box>

                <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                  <Chip
                    label={item.query_type}
                    size="small"
                    color={getQueryTypeColor(item.query_type)}
                  />
                  {item.execution_time_ms && (
                    <Chip
                      label={`${item.execution_time_ms}ms`}
                      size="small"
                      variant="outlined"
                    />
                  )}
                  {item.result_count !== null && (
                    <Chip
                      label={`${item.result_count} rows`}
                      size="small"
                      variant="outlined"
                    />
                  )}
                  {item.user_feedback && (
                    <Chip
                      label={item.user_feedback}
                      size="small"
                      color={item.user_feedback === 'helpful' ? 'success' : 'error'}
                      variant="outlined"
                    />
                  )}
                </Box>

                <ListItemSecondaryAction>
                  <Tooltip title="Copy SQL">
                    <IconButton
                      edge="end"
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleCopySQL(item.sql)
                      }}
                    >
                      <CopyIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Run Query">
                    <IconButton
                      edge="end"
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleRunQuery(item.sql)
                      }}
                    >
                      <RunIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </ListItemSecondaryAction>
              </ListItem>
              {index < filteredHistory.length - 1 && <Divider />}
            </React.Fragment>
          ))}
        </List>

        {filteredHistory.length === 0 && (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Typography color="text.secondary">
              {searchTerm ? 'No queries match your search' : 'No query history yet'}
            </Typography>
          </Box>
        )}
      </Paper>

      {/* Query Details */}
      {selectedQuery && (
        <Paper sx={{ width: 400, p: 2, overflow: 'auto' }}>
          <Typography variant="h6" sx={{ mb: 2 }}>Query Details</Typography>
          
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" color="text.secondary">Natural Language</Typography>
            <Typography variant="body2">{selectedQuery.natural_language}</Typography>
          </Box>

          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" color="text.secondary">Generated SQL</Typography>
            <Paper
              variant="outlined"
              sx={{
                p: 1,
                bgcolor: 'background.default',
                fontFamily: 'monospace',
                fontSize: 13,
                color: 'primary.main',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}
            >
              {selectedQuery.sql}
            </Paper>
          </Box>

          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" color="text.secondary">Metadata</Typography>
            <List dense>
              <ListItem>
                <ListItemText
                  primary="Query Type"
                  secondary={selectedQuery.query_type}
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="Execution Time"
                  secondary={`${selectedQuery.execution_time_ms || 'N/A'} ms`}
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="Result Count"
                  secondary={selectedQuery.result_count || 'N/A'}
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="Created At"
                  secondary={formatDate(selectedQuery.created_at)}
                />
              </ListItem>
            </List>
          </Box>

          <Box sx={{ display: 'flex', gap: 1, justifyContent: 'center' }}>
            <Button
              variant="outlined"
              startIcon={<ThumbUpIcon />}
              onClick={() => handleFeedback(selectedQuery.id, 'helpful')}
              color="success"
            >
              Helpful
            </Button>
            <Button
              variant="outlined"
              startIcon={<ThumbDownIcon />}
              onClick={() => handleFeedback(selectedQuery.id, 'not_helpful')}
              color="error"
            >
              Not Helpful
            </Button>
          </Box>
        </Paper>
      )}
    </Box>
  )
}

export default QueryHistory