import { useState } from 'react'

function Settings() {
  const [settings, setSettings] = useState({
    theme: 'light',
    autoSave: true,
    autoValidate: true,
    defaultFormat: 'svg',
    maxGraphSize: 10000,
  })
  
  const handleChange = (key: string, value: any) => {
    setSettings(prev => ({ ...prev, [key]: value }))
  }
  
  const handleSave = () => {
    localStorage.setItem('settings', JSON.stringify(settings))
    alert('Settings saved!')
  }
  
  return (
    <div className="settings">
      <div className="page-header">
        <h1>Settings</h1>
      </div>
      
      <div className="settings-sections">
        <section className="settings-section">
          <h2>Appearance</h2>
          
          <div className="setting-item">
            <label htmlFor="theme">Theme</label>
            <select
              id="theme"
              className="form-input"
              value={settings.theme}
              onChange={(e) => handleChange('theme', e.target.value)}
            >
              <option value="light">Light</option>
              <option value="dark">Dark</option>
              <option value="auto">Auto</option>
            </select>
          </div>
        </section>
        
        <section className="settings-section">
          <h2>Editor</h2>
          
          <div className="setting-item">
            <label>
              <input
                type="checkbox"
                checked={settings.autoSave}
                onChange={(e) => handleChange('autoSave', e.target.checked)}
              />
              Auto-save graphs
            </label>
          </div>
          
          <div className="setting-item">
            <label>
              <input
                type="checkbox"
                checked={settings.autoValidate}
                onChange={(e) => handleChange('autoValidate', e.target.checked)}
              />
              Auto-validate on changes
            </label>
          </div>
          
          <div className="setting-item">
            <label htmlFor="defaultFormat">Default export format</label>
            <select
              id="defaultFormat"
              className="form-input"
              value={settings.defaultFormat}
              onChange={(e) => handleChange('defaultFormat', e.target.value)}
            >
              <option value="svg">SVG</option>
              <option value="png">PNG</option>
              <option value="html">HTML</option>
            </select>
          </div>
        </section>
        
        <section className="settings-section">
          <h2>Performance</h2>
          
          <div className="setting-item">
            <label htmlFor="maxGraphSize">Max graph size (nodes)</label>
            <input
              id="maxGraphSize"
              type="number"
              className="form-input"
              value={settings.maxGraphSize}
              onChange={(e) => handleChange('maxGraphSize', parseInt(e.target.value))}
            />
          </div>
        </section>
        
        <div className="settings-actions">
          <button className="btn btn-primary" onClick={handleSave}>
            Save Settings
          </button>
          <button className="btn btn-outline">
            Reset to Defaults
          </button>
        </div>
      </div>
    </div>
  )
}

export default Settings