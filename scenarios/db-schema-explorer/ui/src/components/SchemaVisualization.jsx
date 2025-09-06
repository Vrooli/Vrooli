import React, { useCallback, useMemo } from 'react'
import ReactFlow, {
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
} from 'react-flow-renderer'
import { Box, Typography, CircularProgress, Paper, Chip } from '@mui/material'

const TableNode = ({ data }) => {
  return (
    <Paper
      sx={{
        p: 1,
        minWidth: 200,
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: 'primary.main',
      }}
    >
      <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
        {data.label}
      </Typography>
      <Box sx={{ maxHeight: 200, overflow: 'auto' }}>
        {data.columns?.slice(0, 5).map((col, idx) => (
          <Box key={idx} sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
            <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
              {col.name}
            </Typography>
            <Chip label={col.type} size="small" variant="outlined" />
          </Box>
        ))}
        {data.columns?.length > 5 && (
          <Typography variant="caption" color="text.secondary">
            ... and {data.columns.length - 5} more columns
          </Typography>
        )}
      </Box>
    </Paper>
  )
}

const nodeTypes = {
  table: TableNode,
}

function SchemaVisualization({ schemaData, database, onRefresh }) {
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  // Convert schema data to React Flow nodes and edges
  useMemo(() => {
    if (!schemaData?.tables) return

    const newNodes = []
    const newEdges = []
    const tablePositions = {}

    // Create nodes for each table
    schemaData.tables.forEach((table, index) => {
      const x = (index % 4) * 300
      const y = Math.floor(index / 4) * 250
      
      tablePositions[table.name] = { x, y }
      
      newNodes.push({
        id: table.name,
        type: 'table',
        position: { x, y },
        data: {
          label: table.name,
          columns: table.columns || [],
        },
      })
    })

    // Create edges for relationships
    if (schemaData.relationships) {
      schemaData.relationships.forEach((rel, index) => {
        newEdges.push({
          id: `e${index}`,
          source: rel.from_table,
          target: rel.to_table,
          label: rel.from_column + ' → ' + rel.to_column,
          type: 'smoothstep',
          animated: true,
          style: { stroke: '#00ff88' },
        })
      })
    }

    setNodes(newNodes)
    setEdges(newEdges)
  }, [schemaData])

  const onConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  )

  if (!schemaData) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box sx={{ width: '100%', height: '100%' }}>
      <Box sx={{ mb: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6">
          Database Schema: {database}
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Chip label={`${schemaData.tables?.length || 0} Tables`} />
          <Chip label={`${schemaData.relationships?.length || 0} Relations`} />
        </Box>
      </Box>
      
      <Box sx={{ height: 'calc(100% - 50px)', border: '1px solid', borderColor: 'divider' }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          nodeTypes={nodeTypes}
          fitView
        >
          <Controls />
          <MiniMap
            nodeColor={(node) => '#00ff88'}
            style={{
              backgroundColor: '#1a1a1a',
            }}
          />
          <Background variant="dots" gap={12} size={1} />
        </ReactFlow>
      </Box>
    </Box>
  )
}

export default SchemaVisualization