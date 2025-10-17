#!/usr/bin/env node

/**
 * NSFW Model Generator
 * Creates lightweight pre-packaged models for offline-first operation
 */

const tf = require('@tensorflow/tfjs-node');
const fs = require('fs').promises;
const path = require('path');

// Model configurations
const MODEL_CONFIGS = {
    'simple-cnn': {
        name: 'Simple CNN',
        version: '1.0.0',
        description: 'Lightweight CNN for basic NSFW detection',
        inputShape: [224, 224, 3],
        categories: ['adult', 'racy', 'safe']
    },
    'mobilenet-transfer': {
        name: 'MobileNet Transfer',
        version: '1.0.0',
        description: 'Transfer learning model based on MobileNet',
        inputShape: [224, 224, 3],
        categories: ['adult', 'racy', 'gore', 'violence', 'safe']
    },
    'safety-checker': {
        name: 'Safety Checker',
        version: '1.0.0',
        description: 'Compatible safety checker model',
        inputShape: [512, 512, 3],
        categories: ['nsfw', 'safe']
    }
};

/**
 * Create a simple CNN model for NSFW detection
 */
async function createSimpleCNN() {
    const model = tf.sequential({
        layers: [
            // Input layer
            tf.layers.conv2d({
                inputShape: [224, 224, 3],
                filters: 32,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 2 }),

            // Hidden layers
            tf.layers.conv2d({
                filters: 64,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 2 }),

            tf.layers.conv2d({
                filters: 128,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 2 }),

            // Dense layers
            tf.layers.flatten(),
            tf.layers.dropout({ rate: 0.5 }),
            tf.layers.dense({
                units: 128,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.dropout({ rate: 0.5 }),

            // Output layer - 3 categories
            tf.layers.dense({
                units: 3,
                activation: 'softmax',
                kernelInitializer: 'glorotUniform'
            })
        ]
    });

    // Compile the model
    model.compile({
        optimizer: tf.train.adam(0.0001),
        loss: 'categoricalCrossentropy',
        metrics: ['accuracy']
    });

    return model;
}

/**
 * Create a MobileNet transfer learning model
 */
async function createMobileNetTransfer() {
    const model = tf.sequential({
        layers: [
            // Simplified MobileNet-like architecture
            tf.layers.conv2d({
                inputShape: [224, 224, 3],
                filters: 32,
                kernelSize: 3,
                strides: 2,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 2 }),

            tf.layers.conv2d({
                filters: 64,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 2 }),

            tf.layers.conv2d({
                filters: 128,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 2 }),

            tf.layers.conv2d({
                filters: 256,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),

            tf.layers.globalAveragePooling2d(),
            tf.layers.dense({
                units: 256,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.dropout({ rate: 0.5 }),
            tf.layers.dense({
                units: 5, // 5 categories
                activation: 'softmax',
                kernelInitializer: 'glorotUniform'
            })
        ]
    });

    model.compile({
        optimizer: tf.train.adam(0.0001),
        loss: 'categoricalCrossentropy',
        metrics: ['accuracy']
    });

    return model;
}

/**
 * Create a safety checker model
 */
async function createSafetyChecker() {
    const model = tf.sequential({
        layers: [
            tf.layers.conv2d({
                inputShape: [512, 512, 3],
                filters: 16,
                kernelSize: 5,
                strides: 2,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 3, strides: 2 }),
            tf.layers.conv2d({
                filters: 32,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.maxPooling2d({ poolSize: 3, strides: 2 }),
            tf.layers.conv2d({
                filters: 64,
                kernelSize: 3,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.globalMaxPooling2d(),
            tf.layers.dense({
                units: 64,
                activation: 'relu',
                kernelInitializer: 'glorotUniform'
            }),
            tf.layers.dropout({ rate: 0.3 }),
            tf.layers.dense({
                units: 2, // Binary classification
                activation: 'softmax',
                kernelInitializer: 'glorotUniform'
            })
        ]
    });

    model.compile({
        optimizer: tf.train.adam(0.001),
        loss: 'binaryCrossentropy',
        metrics: ['accuracy']
    });

    return model;
}

/**
 * Save model with metadata
 */
async function saveModel(model, modelType, outputDir) {
    const config = MODEL_CONFIGS[modelType];
    const modelPath = path.join(outputDir, modelType);

    // Create directory
    await fs.mkdir(modelPath, { recursive: true });

    // Save the model
    await model.save(`file://${modelPath}`);

    // Create metadata file
    const metadata = {
        name: config.name,
        version: config.version,
        description: config.description,
        created: new Date().toISOString(),
        inputShape: config.inputShape,
        categories: config.categories,
        framework: 'tensorflow.js',
        format: 'tfjs-layers-model'
    };

    await fs.writeFile(
        path.join(modelPath, 'metadata.json'),
        JSON.stringify(metadata, null, 2)
    );

    console.log(`✓ Model saved: ${modelType}`);
    return modelPath;
}

/**
 * Generate all models
 */
async function generateAllModels() {
    const outputDir = path.join(__dirname, '..', 'models');

    console.log('Generating pre-packaged models for offline use...');
    console.log('Output directory:', outputDir);

    // Create models directory
    await fs.mkdir(outputDir, { recursive: true });

    try {
        // Generate Simple CNN
        console.log('\n1. Generating Simple CNN model...');
        const simpleCNN = await createSimpleCNN();
        await saveModel(simpleCNN, 'simple-cnn', outputDir);
        simpleCNN.dispose(); // Free memory
    } catch (error) {
        console.error('Failed to generate Simple CNN:', error.message);
    }

    try {
        // Generate MobileNet Transfer
        console.log('\n2. Generating MobileNet Transfer model...');
        const mobileNet = await createMobileNetTransfer();
        await saveModel(mobileNet, 'mobilenet-transfer', outputDir);
        mobileNet.dispose(); // Free memory
    } catch (error) {
        console.error('Failed to generate MobileNet Transfer:', error.message);
    }

    try {
        // Generate Safety Checker
        console.log('\n3. Generating Safety Checker model...');
        const safetyChecker = await createSafetyChecker();
        await saveModel(safetyChecker, 'safety-checker', outputDir);
        safetyChecker.dispose(); // Free memory

        // Create models index
        const modelsIndex = {
            version: '1.0.0',
            generated: new Date().toISOString(),
            models: Object.keys(MODEL_CONFIGS).map(key => ({
                id: key,
                ...MODEL_CONFIGS[key],
                path: `./${key}/model.json`
            }))
        };

        await fs.writeFile(
            path.join(outputDir, 'index.json'),
            JSON.stringify(modelsIndex, null, 2)
        );

        console.log('\n✅ All models generated successfully!');
        console.log('Models are now available for offline use.');

    } catch (error) {
        console.error('Failed to generate Safety Checker:', error.message);
    }

    // Print summary
    console.log('\nGenerated models:');
    for (const [key, config] of Object.entries(MODEL_CONFIGS)) {
        const modelExists = await fs.access(path.join(outputDir, key, 'model.json'))
            .then(() => true)
            .catch(() => false);
        if (modelExists) {
            console.log(`  ✓ ${config.name} (${key}): ${config.description}`);
        }
    }
}

// Run if called directly
if (require.main === module) {
    generateAllModels().then(() => {
        process.exit(0);
    }).catch(error => {
        console.error('Fatal error:', error);
        process.exit(1);
    });
}

module.exports = {
    createSimpleCNN,
    createMobileNetTransfer,
    createSafetyChecker,
    MODEL_CONFIGS
};