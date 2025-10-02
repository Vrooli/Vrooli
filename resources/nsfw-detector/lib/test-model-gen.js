#!/usr/bin/env node

const tf = require('@tensorflow/tfjs-node');
const path = require('path');

async function testModel() {
    console.log('Testing model generation...');

    try {
        // Test simple model
        const model = tf.sequential();

        model.add(tf.layers.conv2d({
            inputShape: [224, 224, 3],
            filters: 32,
            kernelSize: 3,
            strides: 2,
            activation: 'relu'
        }));

        model.add(tf.layers.maxPooling2d({ poolSize: 2 }));

        model.add(tf.layers.flatten());

        model.add(tf.layers.dense({
            units: 5,
            activation: 'softmax'
        }));

        model.compile({
            optimizer: 'adam',
            loss: 'categoricalCrossentropy'
        });

        console.log('Model created successfully');
        model.summary();

        // Try to save it
        const outputPath = path.join(__dirname, '..', 'models', 'test-model');
        await model.save(`file://${outputPath}`);
        console.log('Model saved successfully to:', outputPath);

    } catch (error) {
        console.error('Error:', error.message);
        console.error('Stack:', error.stack);
    }
}

testModel();