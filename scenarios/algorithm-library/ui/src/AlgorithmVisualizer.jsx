import React, { useState, useEffect, useRef } from 'react';
import './AlgorithmVisualizer.css';

const AlgorithmVisualizer = ({ algorithm, onClose }) => {
  const [array, setArray] = useState([]);
  const [comparing, setComparing] = useState([]);
  const [swapping, setSwapping] = useState([]);
  const [sorted, setSorted] = useState([]);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(500); // milliseconds
  const [currentStep, setCurrentStep] = useState(0);
  const [steps, setSteps] = useState([]);
  const animationRef = useRef(null);
  const speedRef = useRef(speed);

  // Update speed ref when speed changes
  useEffect(() => {
    speedRef.current = speed;
  }, [speed]);

  // Initialize array on mount
  useEffect(() => {
    generateRandomArray();
  }, []);

  const generateRandomArray = () => {
    const newArray = Array.from({ length: 10 }, () => Math.floor(Math.random() * 50) + 10);
    setArray(newArray);
    setSorted([]);
    setComparing([]);
    setSwapping([]);
    setCurrentStep(0);
    setSteps(generateSteps(newArray, algorithm?.name || 'bubblesort'));
  };

  const generateSteps = (arr, algorithmName) => {
    const steps = [];
    const workingArray = [...arr];
    
    switch(algorithmName.toLowerCase()) {
      case 'bubblesort':
      case 'bubble_sort':
        for (let i = 0; i < workingArray.length; i++) {
          for (let j = 0; j < workingArray.length - i - 1; j++) {
            // Comparison step
            steps.push({
              array: [...workingArray],
              comparing: [j, j + 1],
              swapping: [],
              sorted: Array.from({ length: i }, (_, idx) => workingArray.length - 1 - idx),
              description: `Comparing elements at positions ${j} and ${j + 1}`
            });
            
            if (workingArray[j] > workingArray[j + 1]) {
              // Swap step
              [workingArray[j], workingArray[j + 1]] = [workingArray[j + 1], workingArray[j]];
              steps.push({
                array: [...workingArray],
                comparing: [],
                swapping: [j, j + 1],
                sorted: Array.from({ length: i }, (_, idx) => workingArray.length - 1 - idx),
                description: `Swapping ${workingArray[j + 1]} and ${workingArray[j]}`
              });
            }
          }
          // Mark element as sorted
          steps.push({
            array: [...workingArray],
            comparing: [],
            swapping: [],
            sorted: Array.from({ length: i + 1 }, (_, idx) => workingArray.length - 1 - idx),
            description: `Element at position ${workingArray.length - 1 - i} is now in correct position`
          });
        }
        break;
        
      case 'quicksort':
      case 'quick_sort':
        const quickSortSteps = [];
        const quickSort = (arr, low, high, sortedIndices = []) => {
          if (low < high) {
            const pivotIndex = low;
            quickSortSteps.push({
              array: [...arr],
              comparing: [pivotIndex],
              swapping: [],
              sorted: sortedIndices,
              description: `Selecting pivot element ${arr[pivotIndex]} at position ${pivotIndex}`
            });
            
            let i = low + 1;
            for (let j = low + 1; j <= high; j++) {
              quickSortSteps.push({
                array: [...arr],
                comparing: [pivotIndex, j],
                swapping: [],
                sorted: sortedIndices,
                description: `Comparing pivot ${arr[pivotIndex]} with element ${arr[j]}`
              });
              
              if (arr[j] < arr[pivotIndex]) {
                [arr[i], arr[j]] = [arr[j], arr[i]];
                quickSortSteps.push({
                  array: [...arr],
                  comparing: [],
                  swapping: [i, j],
                  sorted: sortedIndices,
                  description: `Moving ${arr[i]} to position ${i}`
                });
                i++;
              }
            }
            
            [arr[pivotIndex], arr[i - 1]] = [arr[i - 1], arr[pivotIndex]];
            const partitionIndex = i - 1;
            quickSortSteps.push({
              array: [...arr],
              comparing: [],
              swapping: [pivotIndex, partitionIndex],
              sorted: [...sortedIndices, partitionIndex],
              description: `Pivot ${arr[partitionIndex]} is now in correct position ${partitionIndex}`
            });
            
            quickSort(arr, low, partitionIndex - 1, [...sortedIndices, partitionIndex]);
            quickSort(arr, partitionIndex + 1, high, [...sortedIndices, partitionIndex]);
          }
        };
        
        quickSort([...workingArray], 0, workingArray.length - 1);
        return quickSortSteps;
        
      case 'mergesort':
      case 'merge_sort':
        const mergeSortSteps = [];
        const mergeSort = (arr, left, right, depth = 0) => {
          if (left >= right) return;
          
          const mid = Math.floor((left + right) / 2);
          
          mergeSortSteps.push({
            array: [...arr],
            comparing: Array.from({ length: right - left + 1 }, (_, i) => left + i),
            swapping: [],
            sorted: [],
            description: `Dividing array from position ${left} to ${right}`
          });
          
          mergeSort(arr, left, mid, depth + 1);
          mergeSort(arr, mid + 1, right, depth + 1);
          
          // Merge
          const leftArr = arr.slice(left, mid + 1);
          const rightArr = arr.slice(mid + 1, right + 1);
          let i = 0, j = 0, k = left;
          
          while (i < leftArr.length && j < rightArr.length) {
            mergeSortSteps.push({
              array: [...arr],
              comparing: [left + i, mid + 1 + j],
              swapping: [],
              sorted: [],
              description: `Comparing ${leftArr[i]} and ${rightArr[j]}`
            });
            
            if (leftArr[i] <= rightArr[j]) {
              arr[k] = leftArr[i];
              i++;
            } else {
              arr[k] = rightArr[j];
              j++;
            }
            
            mergeSortSteps.push({
              array: [...arr],
              comparing: [],
              swapping: [k],
              sorted: [],
              description: `Placing ${arr[k]} at position ${k}`
            });
            k++;
          }
          
          while (i < leftArr.length) {
            arr[k] = leftArr[i];
            mergeSortSteps.push({
              array: [...arr],
              comparing: [],
              swapping: [k],
              sorted: [],
              description: `Placing remaining ${arr[k]} at position ${k}`
            });
            i++;
            k++;
          }
          
          while (j < rightArr.length) {
            arr[k] = rightArr[j];
            mergeSortSteps.push({
              array: [...arr],
              comparing: [],
              swapping: [k],
              sorted: [],
              description: `Placing remaining ${arr[k]} at position ${k}`
            });
            j++;
            k++;
          }
        };
        
        mergeSort([...workingArray], 0, workingArray.length - 1);
        return mergeSortSteps;
        
      default:
        // Default: just show the array
        return [{
          array: workingArray,
          comparing: [],
          swapping: [],
          sorted: [],
          description: 'Visualization not available for this algorithm'
        }];
    }
    
    // Add final sorted state
    steps.push({
      array: [...workingArray],
      comparing: [],
      swapping: [],
      sorted: Array.from({ length: workingArray.length }, (_, i) => i),
      description: 'Array is now completely sorted!'
    });
    
    return steps;
  };

  const play = () => {
    if (currentStep >= steps.length - 1) {
      setCurrentStep(0);
    }
    setIsPlaying(true);
  };

  const pause = () => {
    setIsPlaying(false);
  };

  const reset = () => {
    setIsPlaying(false);
    setCurrentStep(0);
    if (steps[0]) {
      setArray(steps[0].array);
      setComparing(steps[0].comparing);
      setSwapping(steps[0].swapping);
      setSorted(steps[0].sorted);
    }
  };

  const stepForward = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  };

  const stepBackward = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  // Handle animation
  useEffect(() => {
    if (isPlaying && currentStep < steps.length) {
      animationRef.current = setTimeout(() => {
        if (currentStep < steps.length - 1) {
          setCurrentStep(prev => prev + 1);
        } else {
          setIsPlaying(false);
        }
      }, speedRef.current);
    }
    
    return () => {
      if (animationRef.current) {
        clearTimeout(animationRef.current);
      }
    };
  }, [isPlaying, currentStep, steps.length]);

  // Update visualization when step changes
  useEffect(() => {
    if (steps[currentStep]) {
      setArray(steps[currentStep].array);
      setComparing(steps[currentStep].comparing);
      setSwapping(steps[currentStep].swapping);
      setSorted(steps[currentStep].sorted);
    }
  }, [currentStep, steps]);

  const getBarColor = (index) => {
    if (sorted.includes(index)) return '#00ff00';
    if (swapping.includes(index)) return '#ff0000';
    if (comparing.includes(index)) return '#ffff00';
    return '#00ffff';
  };

  const maxValue = Math.max(...array, 60);

  return (
    <div className="visualizer-modal">
      <div className="visualizer-content">
        <div className="visualizer-header">
          <h2>ALGORITHM VISUALIZATION: {algorithm?.display_name || 'Algorithm'}</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>
        
        <div className="visualization-area">
          <div className="bars-container">
            {array.map((value, index) => (
              <div
                key={index}
                className="bar-wrapper"
                style={{ flex: 1 }}
              >
                <div
                  className="bar"
                  style={{
                    height: `${(value / maxValue) * 100}%`,
                    backgroundColor: getBarColor(index),
                    transition: 'all 0.3s ease'
                  }}
                >
                  <span className="bar-value">{value}</span>
                </div>
                <span className="bar-index">{index}</span>
              </div>
            ))}
          </div>
          
          <div className="step-description">
            {steps[currentStep]?.description || 'Ready to start'}
          </div>
        </div>
        
        <div className="controls">
          <div className="control-buttons">
            <button onClick={generateRandomArray} disabled={isPlaying}>
              NEW ARRAY
            </button>
            <button onClick={stepBackward} disabled={isPlaying || currentStep === 0}>
              ◄◄
            </button>
            {!isPlaying ? (
              <button onClick={play} className="play-btn">
                ► PLAY
              </button>
            ) : (
              <button onClick={pause} className="pause-btn">
                ❚❚ PAUSE
              </button>
            )}
            <button onClick={stepForward} disabled={isPlaying || currentStep >= steps.length - 1}>
              ►►
            </button>
            <button onClick={reset} disabled={isPlaying}>
              RESET
            </button>
          </div>
          
          <div className="speed-control">
            <label>Speed:</label>
            <input
              type="range"
              min="100"
              max="2000"
              step="100"
              value={2100 - speed}
              onChange={(e) => setSpeed(2100 - parseInt(e.target.value))}
              disabled={isPlaying}
            />
            <span>{speed}ms</span>
          </div>
          
          <div className="progress">
            Step {currentStep + 1} / {steps.length}
          </div>
        </div>
      </div>
    </div>
  );
};

export default AlgorithmVisualizer;