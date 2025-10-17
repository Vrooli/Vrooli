import React from 'react'
import { FormControl, InputLabel, Select, MenuItem } from '@mui/material'

function DatabaseSelector({ value, onChange }) {
  const databases = ['main', 'test', 'development', 'staging', 'production']

  return (
    <FormControl size="small" sx={{ minWidth: 150 }}>
      <InputLabel>Database</InputLabel>
      <Select
        value={value}
        label="Database"
        onChange={(e) => onChange(e.target.value)}
      >
        {databases.map((db) => (
          <MenuItem key={db} value={db}>
            {db}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  )
}

export default DatabaseSelector