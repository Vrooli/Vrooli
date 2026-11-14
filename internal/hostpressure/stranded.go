package hostpressure

import "sort"

func Stranded(processes []Process, swapToResident float64) []Process {
	if swapToResident <= 0 {
		swapToResident = 2
	}
	result := make([]Process, 0, len(processes))
	for _, p := range processes {
		if p.Swapped == 0 || float64(p.Swapped) < float64(p.Resident)*swapToResident {
			continue
		}
		result = append(result, p)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Swapped > result[j].Swapped })
	return result
}
