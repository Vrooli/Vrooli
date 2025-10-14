import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { QueryClient, QueryClientProvider } from 'react-query'
import { ThemeProvider, createTheme } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#00ff88',
    },
    secondary: {
      main: '#ff00ff',
    },
    background: {
      default: '#0a0a0a',
      paper: '#1a1a1a',
    },
  },
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
    code: {
      fontFamily: '"Fira Code", "Monaco", "Courier New", monospace',
    },
  },
})


const BRIDGE_FLAG = '__dbSchemaExplorerBridgeInitialized'

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_FLAG]) {
  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[DbSchemaExplorer] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'db-schema-explorer' })
  window[BRIDGE_FLAG] = true
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={darkTheme}>
        <CssBaseline />
        <App />
      </ThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>
)
