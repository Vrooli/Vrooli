import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import LoadingSpinner from '../components/LoadingSpinner'

function PluginGallery() {
  const { data: plugins, isLoading } = useQuery({
    queryKey: ['plugins'],
    queryFn: api.getPlugins,
  })
  
  if (isLoading) {
    return <LoadingSpinner message="Loading plugins..." />
  }
  
  const pluginsByCategory = plugins?.data?.reduce((acc: any, plugin: any) => {
    if (!acc[plugin.category]) {
      acc[plugin.category] = []
    }
    acc[plugin.category].push(plugin)
    return acc
  }, {})
  
  return (
    <div className="plugin-gallery">
      <div className="page-header">
        <h1>Plugin Gallery</h1>
        <p className="page-subtitle">
          Explore available graph types and visualization plugins
        </p>
      </div>
      
      {Object.entries(pluginsByCategory || {}).map(([category, categoryPlugins]: [string, any]) => (
        <section key={category} className="plugin-category">
          <h2 className="category-title">
            {category.charAt(0).toUpperCase() + category.slice(1)}
          </h2>
          
          <div className="plugin-grid">
            {categoryPlugins.map((plugin: any) => (
              <div key={plugin.id} className={`plugin-card ${plugin.enabled ? '' : 'disabled'}`}>
                <div className="plugin-icon" style={{ backgroundColor: plugin.metadata?.color || '#ccc' }}>
                  <span>{plugin.metadata?.icon || '📊'}</span>
                </div>
                
                <div className="plugin-content">
                  <h3>{plugin.name}</h3>
                  <p>{plugin.description}</p>
                  
                  <div className="plugin-formats">
                    <span className="formats-label">Formats:</span>
                    {plugin.formats.map((format: string) => (
                      <span key={format} className="format-badge">{format}</span>
                    ))}
                  </div>
                  
                  <div className="plugin-actions">
                    {plugin.enabled ? (
                      <button className="btn btn-primary btn-sm">Create Graph</button>
                    ) : (
                      <button className="btn btn-outline btn-sm" disabled>Coming Soon</button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}

export default PluginGallery