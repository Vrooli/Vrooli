const express = require('express');
const http = require('http');
const WebSocket = require('ws');
const multer = require('multer');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const rateLimit = require('express-rate-limit');
const fs = require('fs').promises;
const path = require('path');
const { v4: uuidv4 } = require('uuid');
const mime = require('mime-types');
const sharp = require('sharp');
const cron = require('node-cron');
const axios = require('axios');
const { Client } = require('pg');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

// Configuration
const config = {
  port: process.env.API_PORT || 3300,
  uiPort: process.env.UI_PORT || 3301,
  authServiceUrl: process.env.AUTH_SERVICE_URL || 'http://localhost:3250',
  storagePath: process.env.STORAGE_PATH || './data/files',
  maxFileSize: parseInt(process.env.MAX_FILE_SIZE) || 10485760, // 10MB
  defaultExpiryHours: parseInt(process.env.DEFAULT_EXPIRY_HOURS) || 24,
  thumbnailSize: 200
};

// Database setup
const db = new Client({
  connectionString: process.env.POSTGRES_URL || 'postgresql://localhost/vrooli'
});

// In-memory storage for WebSocket connections
const connections = new Map();

// Middleware
app.use(helmet({
  crossOriginEmbedderPolicy: false,
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "blob:"],
      connectSrc: ["'self'", "ws:", "wss:"]
    }
  }
}));

app.use(compression());
app.use(cors({
  origin: [`http://localhost:${config.uiPort}`, 'http://localhost:3000'],
  credentials: true
}));

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100 // limit each IP to 100 requests per windowMs
});
app.use(limiter);

// File upload rate limiting (more restrictive)
const uploadLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 10 // limit each IP to 10 file uploads per minute
});

app.use(express.json({ limit: '1mb' }));
app.use(express.urlencoded({ extended: true }));

// Ensure storage directory exists
async function ensureStorageDirectory() {
  try {
    await fs.mkdir(config.storagePath, { recursive: true });
    await fs.mkdir(path.join(config.storagePath, 'thumbnails'), { recursive: true });
  } catch (error) {
    console.error('Failed to create storage directory:', error);
  }
}

// Authentication middleware
async function authenticate(req, res, next) {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (!token) {
      return res.status(401).json({ error: 'No authentication token provided' });
    }

    const response = await axios.get(`${config.authServiceUrl}/api/v1/auth/validate`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });

    if (response.data.valid) {
      req.user = {
        id: response.data.user_id,
        email: response.data.email,
        roles: response.data.roles || []
      };
      next();
    } else {
      return res.status(401).json({ error: 'Invalid authentication token' });
    }
  } catch (error) {
    console.error('Authentication error:', error.message);
    return res.status(401).json({ error: 'Authentication service unavailable' });
  }
}

// Database operations
const dbQueries = {
  async createSyncItem(item) {
    const query = `
      INSERT INTO sync_items (id, user_id, filename, mime_type, file_size, content_type, storage_path, thumbnail_path, metadata, expires_at)
      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
      RETURNING *
    `;
    const values = [
      item.id, item.user_id, item.filename, item.mime_type, item.file_size,
      item.content_type, item.storage_path, item.thumbnail_path, 
      JSON.stringify(item.metadata), item.expires_at
    ];
    const result = await db.query(query, values);
    return result.rows[0];
  },

  async getSyncItems(userId) {
    const query = `
      SELECT * FROM sync_items 
      WHERE user_id = $1 AND expires_at > NOW()
      ORDER BY created_at DESC
    `;
    const result = await db.query(query, [userId]);
    return result.rows;
  },

  async getSyncItem(id, userId) {
    const query = `
      SELECT * FROM sync_items 
      WHERE id = $1 AND user_id = $2 AND expires_at > NOW()
    `;
    const result = await db.query(query, [id, userId]);
    return result.rows[0];
  },

  async deleteSyncItem(id, userId) {
    const query = `
      DELETE FROM sync_items 
      WHERE id = $1 AND user_id = $2
      RETURNING *
    `;
    const result = await db.query(query, [id, userId]);
    return result.rows[0];
  },

  async getExpiredItems() {
    const query = `
      SELECT * FROM sync_items 
      WHERE expires_at <= NOW()
    `;
    const result = await db.query(query);
    return result.rows;
  },

  async deleteExpiredItems() {
    const query = `
      DELETE FROM sync_items 
      WHERE expires_at <= NOW()
      RETURNING *
    `;
    const result = await db.query(query);
    return result.rows;
  },

  async recordDeviceSession(session) {
    const query = `
      INSERT INTO device_sessions (id, user_id, device_info, websocket_id, last_seen)
      VALUES ($1, $2, $3, $4, NOW())
      ON CONFLICT (websocket_id) 
      DO UPDATE SET last_seen = NOW()
      RETURNING *
    `;
    const values = [session.id, session.user_id, JSON.stringify(session.device_info), session.websocket_id];
    const result = await db.query(query, values);
    return result.rows[0];
  },

  async removeDeviceSession(websocketId) {
    const query = `DELETE FROM device_sessions WHERE websocket_id = $1`;
    await db.query(query, [websocketId]);
  }
};

// File operations
async function generateThumbnail(filePath, outputPath, mimeType) {
  if (!mimeType.startsWith('image/')) {
    return null;
  }

  try {
    await sharp(filePath)
      .resize(config.thumbnailSize, config.thumbnailSize, {
        fit: 'inside',
        withoutEnlargement: true
      })
      .jpeg({ quality: 80 })
      .toFile(outputPath);
    return outputPath;
  } catch (error) {
    console.error('Thumbnail generation failed:', error);
    return null;
  }
}

// WebSocket connection handling
wss.on('connection', (ws, req) => {
  const wsId = uuidv4();
  ws.id = wsId;
  
  console.log(`WebSocket connection established: ${wsId}`);

  ws.on('message', async (message) => {
    try {
      const data = JSON.parse(message);
      
      if (data.type === 'auth') {
        // Authenticate WebSocket connection
        try {
          const response = await axios.get(`${config.authServiceUrl}/api/v1/auth/validate`, {
            headers: { 'Authorization': `Bearer ${data.token}` }
          });

          if (response.data.valid) {
            ws.userId = response.data.user_id;
            connections.set(wsId, { ws, userId: response.data.user_id });
            
            // Record device session
            await dbQueries.recordDeviceSession({
              id: uuidv4(),
              user_id: response.data.user_id,
              device_info: data.device_info || {},
              websocket_id: wsId
            });

            ws.send(JSON.stringify({ type: 'auth_success', userId: response.data.user_id }));
          } else {
            ws.send(JSON.stringify({ type: 'auth_error', message: 'Invalid token' }));
            ws.close();
          }
        } catch (error) {
          console.error('WebSocket auth error:', error);
          ws.send(JSON.stringify({ type: 'auth_error', message: 'Authentication failed' }));
          ws.close();
        }
      }
    } catch (error) {
      console.error('WebSocket message error:', error);
    }
  });

  ws.on('close', async () => {
    console.log(`WebSocket connection closed: ${wsId}`);
    connections.delete(wsId);
    await dbQueries.removeDeviceSession(wsId);
  });

  ws.on('error', (error) => {
    console.error('WebSocket error:', error);
  });
});

// Broadcast to user's connections
function broadcastToUser(userId, message) {
  for (const [wsId, connection] of connections.entries()) {
    if (connection.userId === userId && connection.ws.readyState === WebSocket.OPEN) {
      connection.ws.send(JSON.stringify(message));
    }
  }
}

// Multer configuration for file uploads
const upload = multer({
  dest: config.storagePath,
  limits: {
    fileSize: config.maxFileSize
  },
  fileFilter: (req, file, cb) => {
    // Allow all file types but log potentially dangerous ones
    const dangerousTypes = ['.exe', '.bat', '.cmd', '.scr'];
    const ext = path.extname(file.originalname).toLowerCase();
    
    if (dangerousTypes.includes(ext)) {
      console.warn(`Potentially dangerous file uploaded: ${file.originalname}`);
    }
    
    cb(null, true);
  }
});

// API Routes

// Health check
app.get('/health', (req, res) => {
  res.json({ 
    healthy: true, 
    service: 'device-sync-hub',
    version: '1.0.0',
    timestamp: new Date().toISOString()
  });
});

// Upload file or text
app.post('/api/v1/sync/upload', uploadLimiter, authenticate, upload.single('file'), async (req, res) => {
  try {
    let item;
    const itemId = uuidv4();
    const expiresIn = parseInt(req.body.expires_in) || config.defaultExpiryHours;
    const expiresAt = new Date(Date.now() + expiresIn * 60 * 60 * 1000);

    if (req.file) {
      // File upload
      const originalName = req.file.originalname;
      const mimeType = mime.lookup(originalName) || req.file.mimetype;
      const newFilename = `${itemId}_${originalName}`;
      const newPath = path.join(config.storagePath, newFilename);
      
      // Move file to final location
      await fs.rename(req.file.path, newPath);

      // Generate thumbnail if image
      let thumbnailPath = null;
      if (mimeType.startsWith('image/')) {
        thumbnailPath = path.join(config.storagePath, 'thumbnails', `${itemId}.jpg`);
        await generateThumbnail(newPath, thumbnailPath, mimeType);
      }

      item = {
        id: itemId,
        user_id: req.user.id,
        filename: originalName,
        mime_type: mimeType,
        file_size: req.file.size,
        content_type: 'file',
        storage_path: newPath,
        thumbnail_path: thumbnailPath,
        metadata: {
          original_filename: originalName,
          uploaded_at: new Date().toISOString()
        },
        expires_at: expiresAt
      };
    } else if (req.body.text) {
      // Text/clipboard upload
      const textFilename = `text_${itemId}.txt`;
      const textPath = path.join(config.storagePath, textFilename);
      
      await fs.writeFile(textPath, req.body.text, 'utf8');

      item = {
        id: itemId,
        user_id: req.user.id,
        filename: req.body.content_type === 'clipboard' ? 'Clipboard Content' : 'Text Note',
        mime_type: 'text/plain',
        file_size: Buffer.byteLength(req.body.text, 'utf8'),
        content_type: req.body.content_type || 'text',
        storage_path: textPath,
        thumbnail_path: null,
        metadata: {
          text_preview: req.body.text.substring(0, 100)
        },
        expires_at: expiresAt
      };
    } else {
      return res.status(400).json({ error: 'No file or text provided' });
    }

    // Save to database
    const savedItem = await dbQueries.createSyncItem(item);

    // Broadcast to user's other devices
    broadcastToUser(req.user.id, {
      type: 'item_added',
      item: {
        id: savedItem.id,
        filename: savedItem.filename,
        content_type: savedItem.content_type,
        file_size: savedItem.file_size,
        mime_type: savedItem.mime_type,
        thumbnail_url: savedItem.thumbnail_path ? `/api/v1/sync/thumbnails/${savedItem.id}` : null,
        expires_at: savedItem.expires_at,
        created_at: savedItem.created_at
      },
      timestamp: new Date().toISOString()
    });

    res.json({
      success: true,
      item_id: savedItem.id,
      expires_at: savedItem.expires_at,
      thumbnail_url: savedItem.thumbnail_path ? `/api/v1/sync/thumbnails/${savedItem.id}` : null
    });

  } catch (error) {
    console.error('Upload error:', error);
    res.status(500).json({ error: 'Upload failed' });
  }
});

// List sync items
app.get('/api/v1/sync/items', authenticate, async (req, res) => {
  try {
    const items = await dbQueries.getSyncItems(req.user.id);
    
    const formattedItems = items.map(item => ({
      id: item.id,
      filename: item.filename,
      content_type: item.content_type,
      file_size: item.file_size,
      mime_type: item.mime_type,
      thumbnail_url: item.thumbnail_path ? `/api/v1/sync/thumbnails/${item.id}` : null,
      expires_at: item.expires_at,
      created_at: item.created_at,
      metadata: item.metadata
    }));

    res.json({ items: formattedItems });
  } catch (error) {
    console.error('List items error:', error);
    res.status(500).json({ error: 'Failed to retrieve items' });
  }
});

// Download sync item
app.get('/api/v1/sync/items/:id/download', authenticate, async (req, res) => {
  try {
    const item = await dbQueries.getSyncItem(req.params.id, req.user.id);
    
    if (!item) {
      return res.status(404).json({ error: 'Item not found or expired' });
    }

    // Check if file exists
    try {
      await fs.access(item.storage_path);
    } catch {
      return res.status(404).json({ error: 'File not found on disk' });
    }

    res.setHeader('Content-Disposition', `attachment; filename="${item.filename}"`);
    res.setHeader('Content-Type', item.mime_type);
    res.sendFile(path.resolve(item.storage_path));
  } catch (error) {
    console.error('Download error:', error);
    res.status(500).json({ error: 'Download failed' });
  }
});

// Get thumbnail
app.get('/api/v1/sync/thumbnails/:id', authenticate, async (req, res) => {
  try {
    const item = await dbQueries.getSyncItem(req.params.id, req.user.id);
    
    if (!item || !item.thumbnail_path) {
      return res.status(404).json({ error: 'Thumbnail not found' });
    }

    try {
      await fs.access(item.thumbnail_path);
      res.setHeader('Content-Type', 'image/jpeg');
      res.sendFile(path.resolve(item.thumbnail_path));
    } catch {
      return res.status(404).json({ error: 'Thumbnail file not found on disk' });
    }
  } catch (error) {
    console.error('Thumbnail error:', error);
    res.status(500).json({ error: 'Thumbnail retrieval failed' });
  }
});

// Delete sync item
app.delete('/api/v1/sync/items/:id', authenticate, async (req, res) => {
  try {
    const item = await dbQueries.deleteSyncItem(req.params.id, req.user.id);
    
    if (!item) {
      return res.status(404).json({ error: 'Item not found' });
    }

    // Delete files from disk
    try {
      await fs.unlink(item.storage_path);
      if (item.thumbnail_path) {
        await fs.unlink(item.thumbnail_path);
      }
    } catch (error) {
      console.warn('File deletion warning:', error.message);
    }

    // Broadcast deletion to user's other devices
    broadcastToUser(req.user.id, {
      type: 'item_deleted',
      item_id: item.id,
      timestamp: new Date().toISOString()
    });

    res.json({
      success: true,
      deleted_at: new Date().toISOString()
    });
  } catch (error) {
    console.error('Delete error:', error);
    res.status(500).json({ error: 'Deletion failed' });
  }
});

// Settings endpoints
app.get('/api/v1/sync/settings', authenticate, (req, res) => {
  res.json({
    max_file_size: config.maxFileSize,
    default_expiry_hours: config.defaultExpiryHours,
    thumbnail_size: config.thumbnailSize
  });
});

// Cleanup expired items (cron job)
async function cleanupExpiredItems() {
  try {
    console.log('Running cleanup of expired items...');
    const expiredItems = await dbQueries.getExpiredItems();
    
    for (const item of expiredItems) {
      try {
        await fs.unlink(item.storage_path);
        if (item.thumbnail_path) {
          await fs.unlink(item.thumbnail_path);
        }
      } catch (error) {
        console.warn(`Failed to delete file for expired item ${item.id}:`, error.message);
      }
    }
    
    const deletedItems = await dbQueries.deleteExpiredItems();
    console.log(`Cleaned up ${deletedItems.length} expired items`);
  } catch (error) {
    console.error('Cleanup error:', error);
  }
}

// Schedule cleanup every hour
cron.schedule('0 * * * *', cleanupExpiredItems);

// Initialize database connection and start server
async function startServer() {
  try {
    await db.connect();
    console.log('Connected to PostgreSQL database');
    
    await ensureStorageDirectory();
    console.log('Storage directories created');

    server.listen(config.port, () => {
      console.log(`Device Sync Hub API server running on port ${config.port}`);
      console.log(`WebSocket server ready for connections`);
      console.log(`Health check: http://localhost:${config.port}/health`);
    });
  } catch (error) {
    console.error('Server startup error:', error);
    process.exit(1);
  }
}

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Shutting down gracefully...');
  server.close(() => {
    db.end();
    process.exit(0);
  });
});

process.on('SIGINT', async () => {
  console.log('Shutting down gracefully...');
  server.close(() => {
    db.end();
    process.exit(0);
  });
});

startServer();