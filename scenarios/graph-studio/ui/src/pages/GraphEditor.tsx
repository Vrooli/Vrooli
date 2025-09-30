import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import LoadingSpinner from '../components/LoadingSpinner'

function GraphEditor() {
  const { id } = useParams<{ id: string }>()
  
  const { data: graph, isLoading } = useQuery({
    queryKey: ['graph', id],
    queryFn: () => api.getGraph(id!),
    enabled: !!id,
  })
  
  if (isLoading) {
    return <LoadingSpinner message="Loading graph..." />
  }
  
  if (!graph) {
    return (
      <div className="error-state">
        <h2>Graph not found</h2>
        <p>The graph you're looking for doesn't exist.</p>
      </div>
    )
  }
  
  return (
    <div className="graph-editor">
      <div className="editor-header">
        <h1>{(graph as any).name}</h1>
        <div className="editor-actions">
          <button className="btn btn-outline">Save</button>
          <button className="btn btn-outline">Export</button>
          <button className="btn btn-outline">Validate</button>
          <button className="btn btn-primary">Convert</button>
        </div>
      </div>
      
      <div className="editor-content">
        <div className="editor-sidebar">
          <h3>Properties</h3>
          <div className="property-group">
            <label>Type</label>
            <p>{(graph as any).type}</p>
          </div>
          <div className="property-group">
            <label>Version</label>
            <p>{(graph as any).version || '1.0'}</p>
          </div>
          <div className="property-group">
            <label>Created</label>
            <p>{new Date((graph as any).created_at).toLocaleDateString()}</p>
          </div>
          <div className="property-group">
            <label>Tags</label>
            <div className="tags">
              {(graph as any).tags?.map((tag: string) => (
                <span key={tag} className="tag">{tag}</span>
              ))}
            </div>
          </div>
        </div>
        
        <div className="editor-canvas">
          <div className="canvas-placeholder">
            <svg viewBox="0 0 800 600" fill="none">
              <rect x="100" y="100" width="150" height="80" rx="8" fill="var(--primary-color)" opacity="0.2" stroke="var(--primary-color)" strokeWidth="2" />
              <text x="175" y="145" textAnchor="middle" fill="var(--text-primary)">Start Node</text>
              
              <rect x="550" y="100" width="150" height="80" rx="8" fill="var(--secondary-color)" opacity="0.2" stroke="var(--secondary-color)" strokeWidth="2" />
              <text x="625" y="145" textAnchor="middle" fill="var(--text-primary)">End Node</text>
              
              <rect x="325" y="300" width="150" height="80" rx="8" fill="var(--accent-color)" opacity="0.2" stroke="var(--accent-color)" strokeWidth="2" />
              <text x="400" y="345" textAnchor="middle" fill="var(--text-primary)">Process</text>
              
              <path d="M250 140 L325 340 M475 340 L550 140" stroke="var(--text-secondary)" strokeWidth="2" markerEnd="url(#arrowhead)" />
              
              <defs>
                <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                  <polygon points="0 0, 10 3, 0 6" fill="var(--text-secondary)" />
                </marker>
              </defs>
            </svg>
            <p className="canvas-message">Graph editor coming soon...</p>
          </div>
        </div>
      </div>
    </div>
  )
}

export default GraphEditor