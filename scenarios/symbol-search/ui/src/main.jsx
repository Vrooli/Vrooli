import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

// Error boundary for production
class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  componentDidCatch(error, errorInfo) {
    console.error('Symbol Search Error:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          padding: '2rem',
          textAlign: 'center',
          fontFamily: 'Inter, sans-serif'
        }}>
          <h2>Something went wrong</h2>
          <p style={{ color: '#64748b', marginTop: '1rem' }}>
            Please refresh the page or check the console for details.
          </p>
          <details style={{ 
            marginTop: '1rem', 
            textAlign: 'left',
            maxWidth: '600px',
            margin: '1rem auto',
            background: '#f8fafc',
            padding: '1rem',
            borderRadius: '8px'
          }}>
            <summary>Error Details</summary>
            <pre style={{ 
              fontSize: '12px', 
              color: '#dc2626',
              marginTop: '0.5rem',
              whiteSpace: 'pre-wrap'
            }}>
              {this.state.error?.toString()}
            </pre>
          </details>
        </div>
      )
    }

    return this.props.children
  }
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>,
)