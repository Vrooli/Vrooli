-- Add more language implementations for algorithms

-- JavaScript implementations
INSERT INTO implementations (algorithm_id, language, code, version, validated, last_validation) VALUES
-- QuickSort JavaScript
((SELECT id FROM algorithms WHERE name = 'quicksort'), 'javascript', 
'function quicksort(arr) {
    if (arr.length <= 1) return arr;
    
    const pivot = arr[Math.floor(arr.length / 2)];
    const left = arr.filter(x => x < pivot);
    const middle = arr.filter(x => x === pivot);
    const right = arr.filter(x => x > pivot);
    
    return [...quicksort(left), ...middle, ...quicksort(right)];
}', '1.0', true, NOW()),

-- MergeSort JavaScript
((SELECT id FROM algorithms WHERE name = 'mergesort'), 'javascript',
'function mergesort(arr) {
    if (arr.length <= 1) return arr;
    
    const mid = Math.floor(arr.length / 2);
    const left = mergesort(arr.slice(0, mid));
    const right = mergesort(arr.slice(mid));
    
    return merge(left, right);
}

function merge(left, right) {
    let result = [];
    let i = 0, j = 0;
    
    while (i < left.length && j < right.length) {
        if (left[i] <= right[j]) {
            result.push(left[i++]);
        } else {
            result.push(right[j++]);
        }
    }
    
    return result.concat(left.slice(i)).concat(right.slice(j));
}', '1.0', true, NOW()),

-- BubbleSort JavaScript
((SELECT id FROM algorithms WHERE name = 'bubblesort'), 'javascript',
'function bubblesort(arr) {
    const n = arr.length;
    for (let i = 0; i < n - 1; i++) {
        let swapped = false;
        for (let j = 0; j < n - i - 1; j++) {
            if (arr[j] > arr[j + 1]) {
                [arr[j], arr[j + 1]] = [arr[j + 1], arr[j]];
                swapped = true;
            }
        }
        if (!swapped) break;
    }
    return arr;
}', '1.0', true, NOW()),

-- Binary Search JavaScript
((SELECT id FROM algorithms WHERE name = 'binary_search'), 'javascript',
'function binarysearch(arr, target) {
    let left = 0;
    let right = arr.length - 1;
    
    while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        if (arr[mid] === target) return mid;
        if (arr[mid] < target) {
            left = mid + 1;
        } else {
            right = mid - 1;
        }
    }
    return -1;
}', '1.0', true, NOW()),

-- Go implementations
-- QuickSort Go
((SELECT id FROM algorithms WHERE name = 'quicksort'), 'go',
'func quicksort(arr []int) []int {
    if len(arr) <= 1 {
        return arr
    }
    
    pivot := arr[len(arr)/2]
    var left, middle, right []int
    
    for _, v := range arr {
        if v < pivot {
            left = append(left, v)
        } else if v == pivot {
            middle = append(middle, v)
        } else {
            right = append(right, v)
        }
    }
    
    result := quicksort(left)
    result = append(result, middle...)
    result = append(result, quicksort(right)...)
    return result
}', '1.0', true, NOW()),

-- MergeSort Go
((SELECT id FROM algorithms WHERE name = 'mergesort'), 'go',
'func mergesort(arr []int) []int {
    if len(arr) <= 1 {
        return arr
    }
    
    mid := len(arr) / 2
    left := mergesort(arr[:mid])
    right := mergesort(arr[mid:])
    
    return merge(left, right)
}

func merge(left, right []int) []int {
    result := make([]int, 0, len(left)+len(right))
    i, j := 0, 0
    
    for i < len(left) && j < len(right) {
        if left[i] <= right[j] {
            result = append(result, left[i])
            i++
        } else {
            result = append(result, right[j])
            j++
        }
    }
    
    result = append(result, left[i:]...)
    result = append(result, right[j:]...)
    return result
}', '1.0', true, NOW()),

-- Java implementations  
-- QuickSort Java
((SELECT id FROM algorithms WHERE name = 'quicksort'), 'java',
'public class Solution {
    public int[] quicksort(int[] arr) {
        if (arr.length <= 1) return arr;
        quicksortHelper(arr, 0, arr.length - 1);
        return arr;
    }
    
    private void quicksortHelper(int[] arr, int low, int high) {
        if (low < high) {
            int pi = partition(arr, low, high);
            quicksortHelper(arr, low, pi - 1);
            quicksortHelper(arr, pi + 1, high);
        }
    }
    
    private int partition(int[] arr, int low, int high) {
        int pivot = arr[high];
        int i = low - 1;
        
        for (int j = low; j < high; j++) {
            if (arr[j] < pivot) {
                i++;
                int temp = arr[i];
                arr[i] = arr[j];
                arr[j] = temp;
            }
        }
        
        int temp = arr[i + 1];
        arr[i + 1] = arr[high];
        arr[high] = temp;
        
        return i + 1;
    }
}', '1.0', true, NOW()),

-- C++ implementations
-- QuickSort C++
((SELECT id FROM algorithms WHERE name = 'quicksort'), 'cpp',
'#include <vector>
#include <algorithm>

void quicksort(std::vector<int>& arr, int low, int high) {
    if (low < high) {
        int pi = partition(arr, low, high);
        quicksort(arr, low, pi - 1);
        quicksort(arr, pi + 1, high);
    }
}

int partition(std::vector<int>& arr, int low, int high) {
    int pivot = arr[high];
    int i = low - 1;
    
    for (int j = low; j < high; j++) {
        if (arr[j] < pivot) {
            i++;
            std::swap(arr[i], arr[j]);
        }
    }
    std::swap(arr[i + 1], arr[high]);
    return i + 1;
}

std::vector<int> quicksort(std::vector<int> arr) {
    quicksort(arr, 0, arr.size() - 1);
    return arr;
}', '1.0', true, NOW())

ON CONFLICT DO NOTHING;

-- Additional implementations for insertion_sort and selection_sort
-- Insertion Sort - Python implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated)
VALUES (
    (SELECT id FROM algorithms WHERE name = 'insertion_sort'),
    'python',
    'def insertion_sort(arr):
    """
    Insertion sort - builds sorted array one item at a time
    Time: O(n²) worst/average, O(n) best
    Space: O(1)
    """
    arr = arr.copy()
    n = len(arr)

    for i in range(1, n):
        key = arr[i]
        j = i - 1

        # Move elements greater than key one position ahead
        while j >= 0 and arr[j] > key:
            arr[j + 1] = arr[j]
            j -= 1

        arr[j + 1] = key

    return arr',
    true,
    true
) ON CONFLICT DO NOTHING;

-- Insertion Sort - JavaScript implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated)
VALUES (
    (SELECT id FROM algorithms WHERE name = 'insertion_sort'),
    'javascript',
    'function insertionSort(arr) {
    /*
     * Insertion sort - builds sorted array one item at a time
     * Time: O(n²) worst/average, O(n) best
     * Space: O(1)
     */
    const result = [...arr];
    const n = result.length;

    for (let i = 1; i < n; i++) {
        const key = result[i];
        let j = i - 1;

        // Move elements greater than key one position ahead
        while (j >= 0 && result[j] > key) {
            result[j + 1] = result[j];
            j--;
        }

        result[j + 1] = key;
    }

    return result;
}',
    true,
    true
) ON CONFLICT DO NOTHING;

-- Selection Sort - Python implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated)
VALUES (
    (SELECT id FROM algorithms WHERE name = 'selection_sort'),
    'python',
    'def selection_sort(arr):
    """
    Selection sort - repeatedly finds minimum and swaps to front
    Time: O(n²) in all cases
    Space: O(1)
    """
    arr = arr.copy()
    n = len(arr)

    for i in range(n):
        # Find minimum element in remaining unsorted array
        min_idx = i
        for j in range(i + 1, n):
            if arr[j] < arr[min_idx]:
                min_idx = j

        # Swap the found minimum with the first element
        arr[i], arr[min_idx] = arr[min_idx], arr[i]

    return arr',
    true,
    true
) ON CONFLICT DO NOTHING;

-- Selection Sort - JavaScript implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated)
VALUES (
    (SELECT id FROM algorithms WHERE name = 'selection_sort'),
    'javascript',
    'function selectionSort(arr) {
    /*
     * Selection sort - repeatedly finds minimum and swaps to front
     * Time: O(n²) in all cases
     * Space: O(1)
     */
    const result = [...arr];
    const n = result.length;

    for (let i = 0; i < n; i++) {
        // Find minimum element in remaining unsorted array
        let minIdx = i;
        for (let j = i + 1; j < n; j++) {
            if (result[j] < result[minIdx]) {
                minIdx = j;
            }
        }

        // Swap the found minimum with the first element
        [result[i], result[minIdx]] = [result[minIdx], result[i]];
    }

    return result;
}',
    true,
    true
) ON CONFLICT DO NOTHING;

-- HeapSort - Python implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated)
VALUES (
    (SELECT id FROM algorithms WHERE name = 'heapsort'),
    'python',
    'def heapsort(arr):
    """
    Heap sort - uses heap data structure to sort
    Time: O(n log n) in all cases
    Space: O(1)
    """
    arr = arr.copy()
    n = len(arr)

    def heapify(arr, n, i):
        largest = i
        left = 2 * i + 1
        right = 2 * i + 2

        if left < n and arr[left] > arr[largest]:
            largest = left

        if right < n and arr[right] > arr[largest]:
            largest = right

        if largest != i:
            arr[i], arr[largest] = arr[largest], arr[i]
            heapify(arr, n, largest)

    # Build max heap
    for i in range(n // 2 - 1, -1, -1):
        heapify(arr, n, i)

    # Extract elements one by one
    for i in range(n - 1, 0, -1):
        arr[0], arr[i] = arr[i], arr[0]
        heapify(arr, i, 0)

    return arr',
    true,
    true
) ON CONFLICT DO NOTHING;