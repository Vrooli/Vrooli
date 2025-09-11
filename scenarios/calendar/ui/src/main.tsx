import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import App from './App'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
      retry: 1,
      refetchOnWindowFocus: false
    }
  }
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      <Toaster 
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#1f2937',
            color: '#f3f4f6',
            borderRadius: '0.5rem'
          },
          success: {
            iconTheme: {
              primary: '#10b981',
              secondary: '#f3f4f6'
            }
          },
          error: {
            iconTheme: {
              primary: '#ef4444',
              secondary: '#f3f4f6'
            }
          }
        }}
      />
    </QueryClientProvider>
  </React.StrictMode>
)