import React, { useState, useEffect } from 'react'
import {
  Box,
  AppBar,
  Toolbar,
  Typography,
  IconButton,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Paper,
  Tab,
  Tabs,
  Button,
  CircularProgress,
  Alert,
  Snackbar,
  Divider,
  Chip
} from '@mui/material'
import {
  Menu as MenuIcon,
  Storage as StorageIcon,
  Search as SearchIcon,
  History as HistoryIcon,
  Speed as SpeedIcon,
  AccountTree as AccountTreeIcon,
  Code as CodeIcon,
  PlayArrow as PlayArrowIcon,
  Refresh as RefreshIcon
} from '@mui/icons-material'
import SchemaVisualization from './components/SchemaVisualization'
import QueryBuilder from './components/QueryBuilder'
import QueryHistory from './components/QueryHistory'
import DatabaseSelector from './components/DatabaseSelector'
import { useQuery, useMutation } from 'react-query'
import axios from 'axios'

const API_BASE = '/api/v1'

function App() {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [currentTab, setCurrentTab] = useState(0)
  const [selectedDatabase, setSelectedDatabase] = useState('main')
  const [schemaData, setSchemaData] = useState(null)
  const [notification, setNotification] = useState({ open: false, message: '', severity: 'info' })

  // Health check
  const { data: healthData } = useQuery(
    'health',
    () => axios.get('/health').then(res => res.data),
    { refetchInterval: 30000 }
  )

  // Connect to database
  const connectMutation = useMutation(
    (database) => axios.post(`${API_BASE}/schema/connect`, {
      database_name: database
    }),
    {
      onSuccess: (response) => {
        setSchemaData(response.data)
        showNotification('Connected to database successfully', 'success')
      },
      onError: (error) => {
        showNotification('Failed to connect to database', 'error')
      }
    }
  )

  useEffect(() => {
    // Auto-connect to default database on load
    if (selectedDatabase && !schemaData) {
      connectMutation.mutate(selectedDatabase)
    }
  }, [selectedDatabase])

  const showNotification = (message, severity = 'info') => {
    setNotification({ open: true, message, severity })
  }

  const handleTabChange = (event, newValue) => {
    setCurrentTab(newValue)
  }

  const handleDatabaseChange = (database) => {
    setSelectedDatabase(database)
    setSchemaData(null) // Clear current schema
    connectMutation.mutate(database)
  }

  const handleRefresh = () => {
    if (selectedDatabase) {
      connectMutation.mutate(selectedDatabase)
    }
  }

  const menuItems = [
    { icon: <AccountTreeIcon />, text: 'Schema View', tab: 0 },
    { icon: <SearchIcon />, text: 'Query Builder', tab: 1 },
    { icon: <HistoryIcon />, text: 'Query History', tab: 2 },
    { icon: <SpeedIcon />, text: 'Performance', tab: 3 },
  ]

  return (
    <Box sx={{ display: 'flex', height: '100vh', bgcolor: 'background.default' }}>
      {/* App Bar */}
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
        <Toolbar>
          <IconButton
            color="inherit"
            edge="start"
            onClick={() => setDrawerOpen(!drawerOpen)}
            sx={{ mr: 2 }}
          >
            <MenuIcon />
          </IconButton>
          <StorageIcon sx={{ mr: 2 }} />
          <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
            Database Schema Explorer
          </Typography>
          
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <DatabaseSelector
              value={selectedDatabase}
              onChange={handleDatabaseChange}
            />
            
            <IconButton color="inherit" onClick={handleRefresh}>
              <RefreshIcon />
            </IconButton>
            
            {healthData && (
              <Chip
                label={healthData.status}
                color={healthData.status === 'healthy' ? 'success' : 'error'}
                size="small"
              />
            )}
          </Box>
        </Toolbar>
      </AppBar>

      {/* Drawer */}
      <Drawer
        variant="temporary"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        sx={{
          width: 240,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: 240,
            boxSizing: 'border-box',
            mt: 8,
            bgcolor: 'background.paper'
          },
        }}
      >
        <List>
          {menuItems.map((item, index) => (
            <ListItem
              button
              key={item.text}
              selected={currentTab === item.tab}
              onClick={() => {
                setCurrentTab(item.tab)
                setDrawerOpen(false)
              }}
            >
              <ListItemIcon>{item.icon}</ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItem>
          ))}
        </List>
        
        <Divider />
        
        {schemaData && (
          <Box sx={{ p: 2 }}>
            <Typography variant="caption" color="text.secondary">
              Database Stats
            </Typography>
            <Typography variant="body2">
              Tables: {schemaData.statistics?.total_tables || 0}
            </Typography>
            <Typography variant="body2">
              Columns: {schemaData.statistics?.total_columns || 0}
            </Typography>
            <Typography variant="body2">
              Relations: {schemaData.statistics?.total_relationships || 0}
            </Typography>
          </Box>
        )}
      </Drawer>

      {/* Main Content */}
      <Box component="main" sx={{ flexGrow: 1, p: 3, mt: 8 }}>
        <Paper sx={{ height: 'calc(100vh - 112px)', p: 2 }}>
          <Tabs value={currentTab} onChange={handleTabChange} sx={{ mb: 2 }}>
            <Tab label="Schema Visualization" icon={<AccountTreeIcon />} />
            <Tab label="Query Builder" icon={<CodeIcon />} />
            <Tab label="Query History" icon={<HistoryIcon />} />
            <Tab label="Performance" icon={<SpeedIcon />} />
          </Tabs>

          {/* Tab Panels */}
          <Box sx={{ height: 'calc(100% - 60px)', overflow: 'auto' }}>
            {currentTab === 0 && (
              <SchemaVisualization
                schemaData={schemaData}
                database={selectedDatabase}
                onRefresh={handleRefresh}
              />
            )}
            
            {currentTab === 1 && (
              <QueryBuilder
                database={selectedDatabase}
                onNotification={showNotification}
              />
            )}
            
            {currentTab === 2 && (
              <QueryHistory
                database={selectedDatabase}
                onNotification={showNotification}
              />
            )}
            
            {currentTab === 3 && (
              <Box sx={{ p: 2 }}>
                <Typography variant="h6">Performance Analysis</Typography>
                <Typography color="text.secondary">
                  Query performance metrics and optimization suggestions will appear here.
                </Typography>
              </Box>
            )}
          </Box>
        </Paper>
      </Box>

      {/* Notifications */}
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={() => setNotification({ ...notification, open: false })}
      >
        <Alert
          onClose={() => setNotification({ ...notification, open: false })}
          severity={notification.severity}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </Box>
  )
}

export default App