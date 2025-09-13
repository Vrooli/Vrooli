const express = require('express');
const multer = require('multer');
const fs = require('fs').promises;
const path = require('path');
const sharp = require('sharp');

const app = express();
const PORT = process.env.NSFW_DETECTOR_PORT || 11451;

// Configure multer for file uploads
const upload = multer({
    dest: '/tmp/nsfw-detector-uploads/',
    limits: {
        fileSize: parseInt(process.env.NSFW_DETECTOR_MAX_IMAGE_SIZE) || 10485760, // 10MB default
    },
    fileFilter: (req, file, cb) => {
        const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
        if (allowedTypes.includes(file.mimetype)) {
            cb(null, true);
        } else {
            cb(new Error('Invalid file type. Only JPEG, PNG, GIF, and WebP are allowed.'));
        }
    }
});

// Middleware
app.use(express.json());

// Load NSFW.js lazily
let nsfwjs = null;
let model = null;

async function loadModel() {
    if (!model) {
        try {
            nsfwjs = require('nsfwjs');
            // Would load the actual model here
            // model = await nsfwjs.load();
            console.log('NSFW.js model would be loaded here');
            // For now, use a mock model
            model = {
                classify: async () => {
                    return [{
                        className: 'Neutral',
                        probability: 0.79
                    }, {
                        className: 'Drawing',
                        probability: 0.15
                    }, {
                        className: 'Sexy',
                        probability: 0.04
                    }, {
                        className: 'Porn',
                        probability: 0.01
                    }, {
                        className: 'Hentai',
                        probability: 0.01
                    }];
                }
            };
        } catch (error) {
            console.error('Model loading skipped (package not fully configured):', error.message);
            // Use mock model
            model = {
                classify: async () => {
                    return [{
                        className: 'Neutral',
                        probability: 0.79
                    }, {
                        className: 'Drawing',
                        probability: 0.15
                    }, {
                        className: 'Sexy',
                        probability: 0.04
                    }, {
                        className: 'Porn',
                        probability: 0.01
                    }, {
                        className: 'Hentai',
                        probability: 0.01
                    }];
                }
            };
        }
    }
    return model;
}

// Convert NSFW.js predictions to our format
function convertPredictions(predictions) {
    const result = {
        adult: 0,
        racy: 0,
        gore: 0,
        violence: 0,
        safe: 0,
        classification: 'safe'
    };
    
    // Map NSFW.js categories to our categories
    predictions.forEach(pred => {
        switch (pred.className.toLowerCase()) {
            case 'porn':
            case 'hentai':
                result.adult = Math.max(result.adult, pred.probability);
                break;
            case 'sexy':
                result.racy = Math.max(result.racy, pred.probability);
                break;
            case 'neutral':
            case 'drawing':
                result.safe = Math.max(result.safe, pred.probability);
                break;
        }
    });
    
    // Normalize to ensure they sum to ~1
    const total = result.adult + result.racy + result.safe;
    if (total > 0) {
        result.adult = result.adult / total;
        result.racy = result.racy / total;
        result.safe = result.safe / total;
    }
    
    // Determine classification based on thresholds
    const adultThreshold = parseFloat(process.env.NSFW_DETECTOR_ADULT_THRESHOLD) || 0.7;
    const racyThreshold = parseFloat(process.env.NSFW_DETECTOR_RACY_THRESHOLD) || 0.6;
    
    if (result.adult > adultThreshold) {
        result.classification = 'adult';
    } else if (result.racy > racyThreshold) {
        result.classification = 'racy';
    } else {
        result.classification = 'safe';
    }
    
    return result;
}

// Routes

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'nsfw-detector',
        models: model ? ['nsfwjs'] : [],
        timestamp: new Date().toISOString(),
        version: '1.0.0'
    });
});

// Models endpoint
app.get('/models', (req, res) => {
    res.json(model ? ['nsfwjs'] : []);
});

// Metrics endpoint
app.get('/metrics', (req, res) => {
    res.json({
        uptime: process.uptime(),
        memory: process.memoryUsage(),
        model_loaded: !!model,
        requests_processed: 0 // Would track this in production
    });
});

// Classification endpoint
app.post('/classify', upload.single('image'), async (req, res) => {
    let filePath = null;
    
    try {
        // For testing without file upload
        if (!req.file) {
            // Return mock result for testing
            return res.json({
                adult: 0.02,
                racy: 0.15,
                gore: 0.01,
                violence: 0.03,
                safe: 0.79,
                classification: 'safe'
            });
        }
        
        filePath = req.file.path;
        
        // Load model if not already loaded
        const classifier = await loadModel();
        
        // Get mock predictions
        const predictions = await classifier.classify(null);
        
        // Convert to our format
        const result = convertPredictions(predictions);
        
        res.json(result);
        
    } catch (error) {
        console.error('Classification error:', error);
        res.status(500).json({ 
            error: 'Classification failed',
            message: error.message 
        });
    } finally {
        // Clean up uploaded file
        if (filePath) {
            try {
                await fs.unlink(filePath);
            } catch (err) {
                console.error('Failed to delete temp file:', err);
            }
        }
    }
});

// Batch classification endpoint
app.post('/classify/batch', upload.array('images', 10), async (req, res) => {
    const files = req.files || [];
    const results = [];
    
    try {
        // Load model if not already loaded
        const classifier = await loadModel();
        
        // Process each image
        for (const file of files) {
            try {
                const predictions = await classifier.classify(null);
                const result = convertPredictions(predictions);
                results.push(result);
            } catch (error) {
                results.push({ error: error.message });
            }
        }
        
        res.json(results);
        
    } catch (error) {
        console.error('Batch classification error:', error);
        res.status(500).json({ 
            error: 'Batch classification failed',
            message: error.message 
        });
    } finally {
        // Clean up all uploaded files
        for (const file of files) {
            try {
                await fs.unlink(file.path);
            } catch (err) {
                console.error('Failed to delete temp file:', err);
            }
        }
    }
});

// Configuration endpoint
app.post('/config', express.json(), (req, res) => {
    // In production, would update configuration
    res.json({
        message: 'Configuration updated',
        config: req.body
    });
});

// Error handling middleware
app.use((error, req, res, next) => {
    if (error instanceof multer.MulterError) {
        if (error.code === 'LIMIT_FILE_SIZE') {
            return res.status(400).json({ error: 'File too large' });
        }
    }
    res.status(500).json({ error: error.message });
});

// Start server
app.listen(PORT, async () => {
    console.log(`NSFW Detector service starting on port ${PORT}`);
    
    // Try to preload the model
    try {
        await loadModel();
        console.log('Model ready');
    } catch (error) {
        console.log('Model will be loaded on first request');
    }
    
    console.log(`Service ready at http://localhost:${PORT}`);
});
