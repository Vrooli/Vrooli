import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import LoadingSpinner from '../components/LoadingSpinner'

function GraphList() {
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  
  const { data: graphs, isLoading } = useQuery({
    queryKey: ['graphs', typeFilter],
    queryFn: () => api.getGraphs({ type: typeFilter || undefined }),
  })
  
  const { data: plugins } = useQuery({
    queryKey: ['plugins'],
    queryFn: () => api.getPlugins(),
  })
  
  if (isLoading) {
    return <LoadingSpinner message="Loading graphs..." />
  }
  
  const filteredGraphs = graphs?.data?.filter((graph: any) =>
    graph.name.toLowerCase().includes(filter.toLowerCase()) ||
    graph.description?.toLowerCase().includes(filter.toLowerCase())
  )
  
  return (
    <div className="graph-list">
      <div className="page-header">
        <h1>My Graphs</h1>
        <button className="btn btn-primary">New Graph</button>
      </div>
      
      <div className="filters">
        <input
          type="text"
          placeholder="Search graphs..."
          className="form-input"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        
        <select
          className="form-input"
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
        >
          <option value="">All Types</option>
          {plugins?.data?.map((plugin: any) => (
            <option key={plugin.id} value={plugin.id}>
              {plugin.name}
            </option>
          ))}
        </select>
      </div>
      
      <div className="graph-table">
        {filteredGraphs?.length === 0 ? (
          <div className="empty-state">
            <p>No graphs found</p>
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Updated</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredGraphs?.map((graph: any) => (
                <tr key={graph.id}>
                  <td>
                    <Link to={`/graphs/${graph.id}`}>{graph.name}</Link>
                  </td>
                  <td>{graph.type}</td>
                  <td>{new Date(graph.updated_at).toLocaleDateString()}</td>
                  <td>
                    <div className="actions">
                      <Link to={`/graphs/${graph.id}`} className="btn btn-sm">
                        Edit
                      </Link>
                      <button className="btn btn-sm btn-danger">Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export default GraphList